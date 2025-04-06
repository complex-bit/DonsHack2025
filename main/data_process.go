package main

import (
	"encoding/json"
	"fhshaik/model"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func get_key() string {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	key := os.Getenv("API_KEY")
	if key == "" {
		log.Fatal("API_KEY not found in .env file")
	}

	return key
}

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

// type CanvasAssignment struct {
// 	Name  string `json:"name"`
// 	DueAt string `json:"due_at"`
// 	//CreatedAt    string  `json:"created_at"`
// 	Points       float64 `json:"points_possible"`
// 	HasSubmitted bool    `json:"has_submitted_submissions"`
// }

type CanvasAssignment struct {
	Name              string  `json:"name"`
	DueAt             string  `json:"due_at"`
	CreatedAt         string  `json:"created_at"` // You can also try "posted_at"
	PointsPossible    float64 `json:"points_possible"`
	AssignmentGroupID int     `json:"assignment_group_id"`
	Graded            bool    `json:"graded"`
	Submittable       bool    `json:"can_submit"`
	Locked            bool    `json:"locked_for_user"`

	Submission *struct {
		SubmittedAt   string `json:"submitted_at"`
		WorkflowState string `json:"workflow_state"` // e.g., "submitted", "unsubmitted"
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

		if entries[0].Is_submitted {
			time_ci_prev, _ := time.Parse(layout, entries[i-1].Due_date)
			time_ci, _ := time.Parse(layout, entries[i].Submitted_date)
			completion_durations[i] = (float64(time_ci.UnixNano()) - float64(time_ci_prev.UnixNano())) / 1e9
		} else {
			completion_durations[0] = -1.0
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
		} else {
			//fmt.Println("WHOOOOOOOOOOOOOOOOOOOOOOOOOOOOOO")
		}
	}

	return submitted, unsubmitted
}

func data_chugger_to_model(assignments []assignment) func(float64, float64) float64 {
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

	linear_reg_model := data_chugger_to_model(submitted_assignments)

	today_time := float64(time.Now().UnixNano()) / 1e9
	sort.Slice(unsubmitted_assignments, func(i, j int) bool {
		layout := time.RFC3339
		time_i, _ := time.Parse(layout, assignments[i].entry.Due_date)
		time_j, _ := time.Parse(layout, assignments[j].entry.Due_date)
		today_duration_i := today_time - (float64(time_i.UnixNano()) / 1e9)
		today_duration_j := today_time - (float64(time_j.UnixNano()) / 1e9)
		grade_proportion_i := assignments[i].entry.Points
		grade_proportion_j := assignments[j].entry.Points
		expected_time_i := linear_reg_model(assignments[i].duration, float64(grade_proportion_i))
		expected_time_j := linear_reg_model(assignments[j].duration, float64(grade_proportion_j))
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

func Composite_processor(entries []Entry) {
	// Process the entries through your pipeline
	exits := assignments_to_exits(urgency_sort(entry_processor(entries)))
	fmt.Println(exits)

	// Marshal the slice of structs to JSON
	jsonData, err := json.MarshalIndent(exits, "", "  ")
	if err != nil {
		//fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Write the JSON data to a file
	err = os.WriteFile("exits.json", jsonData, 0644)
	if err != nil {
		//fmt.Println("Error writing to file:", err)
		return
	}

	//fmt.Println("JSON data written to entries.json")
}

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

func getEntries(courseId string, course_name string) []Entry {
	// Replace with your actual values
	canvasToken := get_key() //os.Getenv("CANVAS_TOKEN") // or hardcode for testing
	courseID := courseId
	baseURL := "https://usfca.instructure.com"
	var all_canvas_assign []CanvasAssignment
	url := fmt.Sprintf("%s/api/v1/courses/%s/assignments?include[]=submission", baseURL, courseID)

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil
		}
		req.Header.Set("Authorization", "Bearer "+canvasToken)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil
		}

		// Unmarshal the assignments from the response body
		var assign_entry []CanvasAssignment
		err = json.Unmarshal(body, &assign_entry)
		if err != nil {
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
		if all_canvas_assign[i].Submission.SubmittedAt != "" {
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

func getCourseEntries() []Entry {
	canvasToken := get_key()

	//baseURL := "https://usfca.instructure.com"
	req, _ := http.NewRequest("GET", "https://usfca.instructure.com/api/v1/users/self/courses?include[]=enrollments&enrollment_state=active", nil)
	req.Header.Set("Authorization", "Bearer "+canvasToken)
	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var courses []Course
	err = json.Unmarshal(body, &courses)
	if err != nil {
		panic(err)
	}
	//fmt.Println(courses)
	n := len(courses)
	var entries []Entry
	for i := 0; i < n; i++ {
		entries = append(entries, getEntries(fmt.Sprintf("%d", courses[i].ID), courses[i].Name)...)
	}

	//fmt.Println(entries)
	return entries
	//url := fmt.Sprintf("%s/api/v1/courses?per_page=100", baseURL)

}

func main() {
	Composite_processor(getCourseEntries())
	//getEntries("1624878", "FORTNITE SKIBIDI")
}

// func main() {
// 	// Replace with your actual values
// 	canvasToken := "1018~WyU77kVnVHxLmxLnMkQnPKBYAnJafrzr7XEFJXYUtG8RQDKEhJFNVCyVPhhfD77e" //os.Getenv("CANVAS_TOKEN") // or hardcode for testing
// 	courseID := "1624878"
// 	baseURL := "https://usfca.instructure.com"

// 	url := fmt.Sprintf("%s/api/v1/courses/%s/assignments?include[]=submission", baseURL, courseID)

// 	// Create request
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Set headers
// 	req.Header.Set("Authorization", "Bearer "+canvasToken)

// 	// Send request
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer resp.Body.Close()

// 	// Read body
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		panic(err)
// 	}
// 	////fmt.Println(string(body))
// 	// Parse JSON
// 	var assign_entry []CanvasAssignment
// 	err = json.Unmarshal(body, &assign_entry)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Print assignments
// 	for _, a := range assign_entry {
// 		//fmt.Println(a)
// 		if a.Submission.SubmittedAt == "" {
// 			fmt.Printf("hello")
// 		}
// 		//fmt.Println(a.Submission.SubmittedAt)
// 		fmt.Printf("Name: %s\nDue: %s\nPoints: %.2f\n\n", a.Name, a.DueAt, a.PointsPossible)
// 	}
// }

// func main() {

// 	var apiKey string = get_key()

// 	baseURL := "https://usfca.instructure.com"

// 	courseID := "12345"
// 	url := fmt.Sprintf("%s/api/v1/courses/%s/assignments", baseURL, courseID)
// 	req, err := http.NewRequest("GET", url, nil)

// 	req.Header.Set("Authorization", "Bearer "+apiKey)

// 	jsonFile, err := os.Open("example_struct_fields.json")

// 	if err != nil {
// 		//fmt.Println(err)
// 	}

// 	defer jsonFile.Close()

// 	byteValue, _ := io.ReadAll(jsonFile)

//     var result map[string]interface{}

//     json.Unmarshal([]byte(byteValue), &result)

// 	// //fmt.Println("unmarshalls!")

// 	// Json data from api into struct:
// 	// type Entry struct {
// 	// 	Course_name string
// 	// 	Assign_name string
// 	// 	Due_date string
// 	// 	Submitted_date string
// 	// 	Is_submitted bool
// 	// 	Date_posted string
// 	// 	Category_weight float64
// 	// 	Points int
// 	// }

// 	entries := []Entry{}

// 	newEntry := Entry {
// 		Course_name: result["course_name"].(string),
// 		Assign_name: result["assign_name"].(string),
// 		Due_date: result["due_date"].(string),
// 		Submitted_date: result["submitted_date"].(string),
// 		Category_weight: result["category_weight"].(float64),
// 		Points: int(result["points"].(float64)),
// 	}

// 	entries = append(entries, newEntry)

// 	for i := range(len(entries)) {
// 		entry := entries[i]
// 		value := reflect.ValueOf(entry)
// 		valueType := value.Type()
// 		for j :=  0; j < value.NumField(); j++ {
// 			field := value.Field(j)
// 			fieldName := valueType.Field(j).Name
// 			fieldValue := field.Interface()
// 			//fmt.Println(fieldName, fieldValue)
// 		}
// 	}

// 	Composite_processor(entries)

// }

// func main() {

// 	ent := []Entry{
// 		{
// 			Course_name:     "Math 101",
// 			Assign_name:     "Homework 1",
// 			Due_date:        "2025-04-10T23:59:00Z",
// 			Submitted_date:  "2025-04-09T20:00:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-01T12:00:00Z",
// 			Category_weight: 0.1,
// 			Points:          100,
// 		},
// 		{
// 			Course_name:     "Physics 202",
// 			Assign_name:     "Lab Report 1",
// 			Due_date:        "2025-04-12T23:59:00Z",
// 			Submitted_date:  "2025-04-12T22:30:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-03T10:00:00Z",
// 			Category_weight: 0.15,
// 			Points:          50,
// 		},
// 		{
// 			Course_name:     "CompSci 303",
// 			Assign_name:     "Project Proposal",
// 			Due_date:        "2025-04-15T23:59:00Z",
// 			Submitted_date:  "",
// 			Is_submitted:    false,
// 			Date_posted:     "2025-04-05T15:00:00Z",
// 			Category_weight: 0.2,
// 			Points:          25,
// 		},
// 		{
// 			Course_name:     "Math 101",
// 			Assign_name:     "Homework 2",
// 			Due_date:        "2025-04-18T23:59:00Z",
// 			Submitted_date:  "2025-04-17T19:00:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-10T08:00:00Z",
// 			Category_weight: 0.1,
// 			Points:          100,
// 		},
// 		{
// 			Course_name:     "Physics 202",
// 			Assign_name:     "Problem Set",
// 			Due_date:        "2025-04-20T23:59:00Z",
// 			Submitted_date:  "",
// 			Is_submitted:    false,
// 			Date_posted:     "2025-04-12T11:00:00Z",
// 			Category_weight: 0.1,
// 			Points:          40,
// 		},
// 		{
// 			Course_name:     "CompSci 303",
// 			Assign_name:     "Midterm Report",
// 			Due_date:        "2025-04-22T23:59:00Z",
// 			Submitted_date:  "2025-04-21T23:30:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-14T13:00:00Z",
// 			Category_weight: 0.3,
// 			Points:          60,
// 		},
// 		{
// 			Course_name:     "Math 101",
// 			Assign_name:     "Quiz 1",
// 			Due_date:        "2025-04-24T10:00:00Z",
// 			Submitted_date:  "2025-04-24T09:50:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-18T07:00:00Z",
// 			Category_weight: 0.05,
// 			Points:          20,
// 		},
// 		{
// 			Course_name:     "Physics 202",
// 			Assign_name:     "Lab Report 2",
// 			Due_date:        "2025-04-26T23:59:00Z",
// 			Submitted_date:  "",
// 			Is_submitted:    false,
// 			Date_posted:     "2025-04-19T09:00:00Z",
// 			Category_weight: 0.15,
// 			Points:          50,
// 		},
// 		{
// 			Course_name:     "CompSci 303",
// 			Assign_name:     "Code Review",
// 			Due_date:        "2025-04-28T23:59:00Z",
// 			Submitted_date:  "2025-04-28T18:00:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-20T15:30:00Z",
// 			Category_weight: 0.1,
// 			Points:          30,
// 		},
// 		{
// 			Course_name:     "Math 101",
// 			Assign_name:     "Homework 3",
// 			Due_date:        "2025-05-01T23:59:00Z",
// 			Submitted_date:  "2025-04-30T20:00:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-22T10:00:00Z",
// 			Category_weight: 0.1,
// 			Points:          100,
// 		},
// 		{
// 			Course_name:     "Physics 202",
// 			Assign_name:     "Final Prep",
// 			Due_date:        "2025-05-04T23:59:00Z",
// 			Submitted_date:  "",
// 			Is_submitted:    false,
// 			Date_posted:     "2025-04-26T11:00:00Z",
// 			Category_weight: 0.2,
// 			Points:          70,
// 		},
// 		{
// 			Course_name:     "CompSci 303",
// 			Assign_name:     "Final Project",
// 			Due_date:        "2025-05-08T23:59:00Z",
// 			Submitted_date:  "2025-05-08T22:00:00Z",
// 			Is_submitted:    true,
// 			Date_posted:     "2025-04-28T13:30:00Z",
// 			Category_weight: 0.4,
// 			Points:          100,
// 		},
// 	}
// 	Composite_processor(ent)
// }
