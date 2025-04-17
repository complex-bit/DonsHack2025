package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"server/model"
	"sort"
	"strings"
	"sync"
	"time"
)

// Global cache variables with API key as part of the key
var (
	assignmentCache map[string][]Exit // Map of API key to assignments
	lastCacheUpdate map[string]time.Time
	cacheMutex      sync.RWMutex
	cacheExpiration = 15 * time.Minute // Cache expires after 15 minutes
)

// Session manager instance (defined in auth.go)
var sessionManager *SessionManager

// ----- Entry and Assignment Data Structures -----

type Entry struct {
	Course_name    string `json:"course_name"`
	Assign_name    string `json:"assign_name"`
	Due_date       string `json:"due_date"`
	Submitted_date string `json:"submitted_date"`
	Is_submitted   bool   `json:"is_submitted"`
	Date_posted    string `json:"date_posted"`
	Points         int    `json:"points"`
	Submittable    bool   `json:"can_submit"`
	Graded         bool   `json:"graded"`
	Locked         bool   `json:"locked_for_user"`
}

type CanvasAssignment struct {
	Name              string  `json:"name"`
	DueAt             string  `json:"due_at"`
	CreatedAt         string  `json:"created_at"`
	PointsPossible    float64 `json:"points_possible"`
	AssignmentGroupID int     `json:"assignment_group_id"`
	Graded            bool    `json:"graded"`
	Submittable       bool    `json:"can_submit"`
	Locked            bool    `json:"locked_for_user"`

	Submission *struct {
		SubmittedAt   string `json:"submitted_at"`
		WorkflowState string `json:"workflow_state"`
	} `json:"submission,omitempty"`
}

type assignment struct {
	entry              Entry
	duration           float64
	submitted_duration float64 //-1.0 if not submitted
}

type Exit struct {
	Course_name string `json:"course_name"`
	Assign_name string `json:"assign_name"`
	Due_date    string `json:"due_date"`
	Money       int    `json:"money"`
}

// ----- Data Processing Functions -----

func entry_processor(entries []Entry) (assignments []assignment) {
	sort.Slice(entries, func(i, j int) bool {
		layout := time.RFC3339

		timeI, errI := time.Parse(layout, entries[i].Due_date)
		timeJ, errJ := time.Parse(layout, entries[j].Due_date)

		if errI != nil || errJ != nil {
			// Keep invalid ones at the end
			return errI == nil
		}

		// Sort by earliest due date first (ascending)
		return timeI.Before(timeJ)
	})
	n := len(entries)
	durations := make([]float64, n)
	completion_durations := make([]float64, n)

	if n > 0 {
		layout := time.RFC3339
		time_d0, _ := time.Parse(layout, entries[0].Date_posted)
		time_d1, _ := time.Parse(layout, entries[0].Due_date)

		durations[0] = (float64(time_d1.UnixNano()) - float64(time_d0.UnixNano())) / 1e9

		if entries[0].Is_submitted {
			time_c0, _ := time.Parse(layout, entries[0].Date_posted)
			time_c1, _ := time.Parse(layout, entries[0].Submitted_date)
			completion_durations[0] = (float64(time_c1.UnixNano()) - float64(time_c0.UnixNano())) / 1e9
		} else {
			completion_durations[0] = -1.0
		}

		for i := 1; i < n; i++ {
			time_di_prev, _ := time.Parse(layout, entries[i-1].Due_date)
			time_di, _ := time.Parse(layout, entries[i].Due_date)

			durations[i] = (float64(time_di.UnixNano()) - float64(time_di_prev.UnixNano())) / 1e9

			if entries[i].Is_submitted {
				time_ci_prev, _ := time.Parse(layout, entries[i-1].Due_date)
				time_ci, _ := time.Parse(layout, entries[i].Submitted_date)
				completion_durations[i] = (float64(time_ci.UnixNano()) - float64(time_ci_prev.UnixNano())) / 1e9
			} else {
				completion_durations[i] = -1.0
			}
		}
	}

	assigners := make([]assignment, n)

	for i := 0; i < n; i++ {
		assigners[i] = assignment{
			entry:              entries[i],
			duration:           durations[i],
			submitted_duration: completion_durations[i],
		}
	}

	return assigners
}

func disjoint_assignment_process(assignments []assignment) (submitted_assignments []assignment, unsubmitted_assignments []assignment) {
	var submitted []assignment
	var unsubmitted []assignment
	n := len(assignments)
	for i := 0; i < n; i++ {
		due_time, _ := time.Parse(time.RFC3339, assignments[i].entry.Due_date)
		if assignments[i].entry.Is_submitted {
			submitted = append(submitted, assignments[i])
		} else if time.Now().Before(due_time) {
			unsubmitted = append(unsubmitted, assignments[i])
		}
	}

	return submitted, unsubmitted
}

func data_chugger_to_model(assignments []assignment) func(float64, float64) float64 {
	if len(assignments) == 0 {
		// Return a default model if there are no assignments
		return func(duration, points float64) float64 {
			return duration * 0.5 // Simple default model
		}
	}

	sort.Slice(assignments, func(i, j int) bool {
		layout := time.RFC3339

		timeI, errI := time.Parse(layout, assignments[i].entry.Due_date)
		timeJ, errJ := time.Parse(layout, assignments[j].entry.Due_date)

		if errI != nil || errJ != nil {
			// Keep invalid ones at the end
			return errI == nil
		}

		// Sort by earliest due date first (ascending)
		return timeI.Before(timeJ)
	})

	n := len(assignments)
	durations := make([]float64, n)
	grade_proportions := make([]float64, n)
	completion_durations := make([]float64, n)

	layout := time.RFC3339
	time_d0, _ := time.Parse(layout, assignments[0].entry.Date_posted)
	time_d1, _ := time.Parse(layout, assignments[0].entry.Due_date)

	time_c0, _ := time.Parse(layout, assignments[0].entry.Date_posted)
	time_c1, _ := time.Parse(layout, assignments[0].entry.Submitted_date)

	durations[0] = (float64(time_d1.UnixNano()) - float64(time_d0.UnixNano())) / 1e9
	grade_proportions[0] = float64(assignments[0].entry.Points)
	completion_durations[0] = (float64(time_c1.UnixNano()) - float64(time_c0.UnixNano())) / 1e9

	for i := 1; i < n; i++ {
		time_di_prev, _ := time.Parse(layout, assignments[i-1].entry.Due_date)
		time_di, _ := time.Parse(layout, assignments[i].entry.Due_date)

		time_ci_prev, _ := time.Parse(layout, assignments[i-1].entry.Due_date)
		time_ci, _ := time.Parse(layout, assignments[i].entry.Submitted_date)

		durations[i] = (float64(time_di.UnixNano()) - float64(time_di_prev.UnixNano())) / 1e9
		grade_proportions[i] = float64(assignments[i].entry.Points)
		completion_durations[i] = (float64(time_ci.UnixNano()) - float64(time_ci_prev.UnixNano())) / 1e9
	}

	return model.LinearRegressionModel(durations, grade_proportions, completion_durations)
}

func urgency_sort(assignments []assignment) (unsubmitted []assignment) {
	submitted_assignments, unsubmitted_assignments := disjoint_assignment_process(assignments)

	if len(submitted_assignments) == 0 || len(unsubmitted_assignments) == 0 {
		return unsubmitted_assignments
	}

	linear_reg_model := data_chugger_to_model(submitted_assignments)

	today_time := float64(time.Now().UnixNano()) / 1e9
	sort.Slice(unsubmitted_assignments, func(i, j int) bool {
		layout := time.RFC3339
		time_i, _ := time.Parse(layout, unsubmitted_assignments[i].entry.Due_date)
		time_j, _ := time.Parse(layout, unsubmitted_assignments[j].entry.Due_date)
		today_duration_i := (float64(time_i.UnixNano()) / 1e9) - today_time
		today_duration_j := (float64(time_j.UnixNano()) / 1e9) - today_time
		grade_proportion_i := unsubmitted_assignments[i].entry.Points
		grade_proportion_j := unsubmitted_assignments[j].entry.Points
		expected_time_i := linear_reg_model(unsubmitted_assignments[i].duration, float64(grade_proportion_i))
		expected_time_j := linear_reg_model(unsubmitted_assignments[j].duration, float64(grade_proportion_j))
		return (model.UrgencyDetermination(today_duration_i, expected_time_i, float64(grade_proportion_i)) > model.UrgencyDetermination(today_duration_j, expected_time_j, float64(grade_proportion_j)))
	})

	return unsubmitted_assignments
}

func assignments_to_exits(assignments []assignment) (exits []Exit) {
	n := len(assignments)
	exit_slice := make([]Exit, n)
	cash := 100
	for i := 0; i < n; i++ {
		exit_slice[i].Course_name = assignments[i].entry.Course_name
		exit_slice[i].Assign_name = assignments[i].entry.Assign_name
		exit_slice[i].Due_date = assignments[i].entry.Due_date
		if i < 3 {
			exit_slice[i].Money = cash
			cash = cash / 2
		} else if i == 3 {
			exit_slice[i].Money = 10
		} else {
			exit_slice[i].Money = 5
		}
	}
	return exit_slice
}

// ----- Canvas API Functions -----

func getNextPageURL(linkHeader string) string {
	// Example Link header: <https://api.instructure.com/api/v1/courses/1/assignments?page=2>; rel="next"
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		if strings.Contains(link, `rel="next"`) {
			// Extract URL from the <...> part
			start := strings.Index(link, "<") + 1
			end := strings.Index(link, ">")
			if start > 0 && end > start {
				return link[start:end]
			}
		}
	}
	return ""
}

func getEntries(courseId string, course_name string, apiKey string) []Entry {
	// Use the provided API key instead of getting from .env
	canvasToken := apiKey
	courseID := courseId
	baseURL := "https://usfca.instructure.com"
	var all_canvas_assign []CanvasAssignment
	url := fmt.Sprintf("%s/api/v1/courses/%s/assignments?include[]=submission", baseURL, courseID)

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			return nil
		}
		req.Header.Set("Authorization", "Bearer "+canvasToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error sending request: %v", err)
			return nil
		}
		defer resp.Body.Close()

		// Check for unauthorized status
		if resp.StatusCode == http.StatusUnauthorized {
			log.Printf("Unauthorized access: Invalid API key")
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return nil
		}

		// Unmarshal the assignments from the response body
		var assign_entry []CanvasAssignment
		err = json.Unmarshal(body, &assign_entry)
		if err != nil {
			log.Printf("Error unmarshaling JSON: %v", err)
			return nil
		}

		// Add the assignments to the allAssignments slice
		all_canvas_assign = append(all_canvas_assign, assign_entry...)

		// Check if there's another page of assignments
		linkHeader := resp.Header.Get("Link")
		if linkHeader == "" {
			break // No more pages, exit the loop
		}

		// Parse the Link header to find the next page URL
		nextPage := getNextPageURL(linkHeader)
		if nextPage == "" {
			break // No "next" page, exit the loop
		}

		// Set the URL for the next request
		url = nextPage
	}

	n := len(all_canvas_assign)
	entries := make([]Entry, n)
	for i := 0; i < n; i++ {
		entries[i] = Entry{
			Course_name:    course_name,
			Assign_name:    all_canvas_assign[i].Name,
			Due_date:       all_canvas_assign[i].DueAt,
			Submitted_date: "",
			Is_submitted:   false,
			Date_posted:    all_canvas_assign[i].CreatedAt,
			Points:         int(all_canvas_assign[i].PointsPossible),
			Submittable:    all_canvas_assign[i].Submittable,
			Locked:         all_canvas_assign[i].Locked,
		}
		if all_canvas_assign[i].Submission != nil && all_canvas_assign[i].Submission.SubmittedAt != "" {
			entries[i].Submitted_date = all_canvas_assign[i].Submission.SubmittedAt
			entries[i].Is_submitted = true
		}
	}
	return entries
}

type Course struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func getCourseEntries(apiKey string) []Entry {
	req, _ := http.NewRequest("GET", "https://usfca.instructure.com/api/v1/users/self/courses?include[]=enrollments&enrollment_state=active", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error fetching courses: %v", err)
		return []Entry{}
	}
	defer resp.Body.Close()

	// Check for unauthorized status
	if resp.StatusCode == http.StatusUnauthorized {
		log.Printf("Unauthorized access: Invalid API key")
		return []Entry{}
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		return []Entry{}
	}

	var courses []Course
	err = json.Unmarshal(body, &courses)
	if err != nil {
		log.Printf("Error unmarshaling courses: %v", err)
		return []Entry{}
	}

	n := len(courses)
	var entries []Entry
	for i := 0; i < n; i++ {
		courseEntries := getEntries(fmt.Sprintf("%d", courses[i].ID), courses[i].Name, apiKey)
		if courseEntries != nil {
			entries = append(entries, courseEntries...)
		}
	}

	return entries
}

func process_assignments(apiKey string) []Exit {
	// If apiKey is empty, return demo data
	if apiKey == "" {
		return getDemoAssignments()
	}

	// Check cache for this specific API key
	cacheMutex.RLock()
	cacheValid := false
	if lastCacheUpdate != nil {
		lastUpdate, exists := lastCacheUpdate[apiKey]
		if exists && !lastUpdate.IsZero() && time.Since(lastUpdate) < cacheExpiration {
			cacheValid = true
		}
	}

	if cacheValid && assignmentCache != nil {
		exits, exists := assignmentCache[apiKey]
		if exists && len(exits) > 0 {
			defer cacheMutex.RUnlock()
			return exits
		}
	}
	cacheMutex.RUnlock()

	// Cache is invalid or empty, fetch new data
	entries := getCourseEntries(apiKey)
	assignments := entry_processor(entries)
	unsubmitted := urgency_sort(assignments)
	exits := assignments_to_exits(unsubmitted)

	// Update cache
	cacheMutex.Lock()
	if assignmentCache == nil {
		assignmentCache = make(map[string][]Exit)
	}
	if lastCacheUpdate == nil {
		lastCacheUpdate = make(map[string]time.Time)
	}
	assignmentCache[apiKey] = exits
	lastCacheUpdate[apiKey] = time.Now()
	cacheMutex.Unlock()

	return exits
}

// Get demo assignments for development or when API key is not available
func getDemoAssignments() []Exit {
	// Demo data for development
	demoEntries := []Entry{
		{
			Course_name:    "Math 101",
			Assign_name:    "Homework 1",
			Due_date:       time.Now().Add(48 * time.Hour).Format(time.RFC3339),
			Submitted_date: "",
			Is_submitted:   false,
			Date_posted:    time.Now().Add(-72 * time.Hour).Format(time.RFC3339),
			Points:         100,
		},
		{
			Course_name:    "Computer Science 202",
			Assign_name:    "Programming Project",
			Due_date:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			Submitted_date: "",
			Is_submitted:   false,
			Date_posted:    time.Now().Add(-96 * time.Hour).Format(time.RFC3339),
			Points:         150,
		},
		{
			Course_name:    "History 155",
			Assign_name:    "Research Paper",
			Due_date:       time.Now().Add(72 * time.Hour).Format(time.RFC3339),
			Submitted_date: "",
			Is_submitted:   false,
			Date_posted:    time.Now().Add(-120 * time.Hour).Format(time.RFC3339),
			Points:         75,
		},
	}
	assignments := entry_processor(demoEntries)
	return assignments_to_exits(assignments)
}

// ----- HTTP Server Functions -----

// CORS middleware to handle preflight requests
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		// Proceed to the next handler
		next.ServeHTTP(w, r)
	})
}

// Check if a user is logged in
func isLoggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return false
	}

	session := sessionManager.GetSession(cookie.Value)
	return session != nil
}

// Get user info if logged in
func getLoggedInUser(r *http.Request) *UserSession {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		return nil
	}

	return sessionManager.GetSession(cookie.Value)
}

func main() {
	// Initialize session manager
	sessionManager = NewSessionManager()

	// Initialize cache maps
	assignmentCache = make(map[string][]Exit)
	lastCacheUpdate = make(map[string]time.Time)

	// API endpoints that require authentication
	http.HandleFunc("/assignments", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get API key from session
		apiKey := sessionManager.ApiKeyFromRequest(r)

		// Check if user is logged in
		if !isLoggedIn(r) {
			// For development, still return demo data
			// In production, you might want to return an error instead
			exits := getDemoAssignments()
			json.NewEncoder(w).Encode(exits)
			return
		}

		// Process assignments with the user's API key
		exits := process_assignments(apiKey)

		if err := json.NewEncoder(w).Encode(exits); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// Test data endpoint with increment functionality
	var testValue = 100
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "POST" {
			// Handle POST request to increment value
			testValue += 5
		}

		data := map[string]interface{}{
			"test": testValue,
		}

		json.NewEncoder(w).Encode(data)
	})

	// Session status endpoint
	http.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		user := getLoggedInUser(r)
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"loggedIn": false,
			})
			return
		}

		// Return user info without API key for security
		json.NewEncoder(w).Encode(map[string]interface{}{
			"loggedIn": true,
			"username": user.Username,
			"since":    user.CreatedAt,
		})
	})

	// Authentication routes
	http.HandleFunc("/login", sessionManager.LoginHandler)
	http.HandleFunc("/logout", sessionManager.LogoutHandler)

	// Static file paths
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("../static"))))

	// Handle the main page - with authentication check fully implemented
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// Check if user is logged in - now properly implemented
		if !isLoggedIn(r) {
			log.Printf("User not logged in, redirecting to login page")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// User is logged in, serve the main.html
		log.Printf("User logged in, serving main page")
		http.ServeFile(w, r, "../static/main.html")
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
