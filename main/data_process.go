package main

import (
	"fhshaik/model"
	"fmt"
	"sort"
	"time"
)

type entry struct {
	course_name     string
	assign_name     string
	due_date        string
	submitted_date  string
	is_submitted    bool
	date_posted     string
	category_weight float64
	points          int
}

type assignment struct {
	entry              entry
	duration           float64
	submitted_duration float64 //-1.0 if not submitted
}

type exit struct {
	course_name string
	assign_name string
	due_date    string
	money       int
}

func entry_processor(entries []entry) (assignments []assignment) {

	sort.Slice(entries, func(i, j int) bool {
		layout := time.RFC3339

		timeI, errI := time.Parse(layout, entries[i].due_date)
		timeJ, errJ := time.Parse(layout, entries[j].due_date)

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
	time_d0, _ := time.Parse(layout, entries[0].date_posted)
	time_d1, _ := time.Parse(layout, entries[0].due_date)

	durations[0] = (float64(time_d1.UnixNano()) - float64(time_d0.UnixNano())) / 1e9

	println("HELLLLOOO")
	println(durations[0])
	if entries[0].is_submitted {
		time_c0, _ := time.Parse(layout, entries[0].date_posted)
		time_c1, _ := time.Parse(layout, entries[0].submitted_date)
		completion_durations[0] = (float64(time_c1.UnixNano()) - float64(time_c0.UnixNano())) / 1e9
	} else {
		completion_durations[0] = -1.0
	}

	for i := 1; i < n; i++ {
		time_di_prev, _ := time.Parse(layout, entries[i-1].due_date)
		time_di, _ := time.Parse(layout, entries[i].due_date)

		durations[i] = (float64(time_di.UnixNano()) - float64(time_di_prev.UnixNano())) / 1e9

		if entries[0].is_submitted {
			time_ci_prev, _ := time.Parse(layout, entries[i-1].due_date)
			time_ci, _ := time.Parse(layout, entries[i].submitted_date)
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
		if assignments[i].entry.is_submitted {
			submitted = append(submitted, assignments[i])
		} else {
			unsubmitted = append(unsubmitted, assignments[i])
		}
	}

	return submitted, unsubmitted
}

func data_chugger_to_model(assignments []assignment) func(float64, float64) float64 {
	sort.Slice(assignments, func(i, j int) bool {
		layout := time.RFC3339

		timeI, errI := time.Parse(layout, assignments[i].entry.due_date)
		timeJ, errJ := time.Parse(layout, assignments[j].entry.due_date)

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
	time_d0, _ := time.Parse(layout, assignments[0].entry.date_posted)
	time_d1, _ := time.Parse(layout, assignments[0].entry.due_date)

	time_c0, _ := time.Parse(layout, assignments[0].entry.date_posted)
	time_c1, _ := time.Parse(layout, assignments[0].entry.submitted_date)

	durations[0] = (float64(time_d1.UnixNano()) - float64(time_d0.UnixNano())) / 1e9
	grade_proportions[0] = float64(assignments[0].entry.points)
	completion_durations[0] = (float64(time_c1.UnixNano()) - float64(time_c0.UnixNano())) / 1e9

	for i := 1; i < n; i++ {
		time_di_prev, _ := time.Parse(layout, assignments[i-1].entry.due_date)
		time_di, _ := time.Parse(layout, assignments[i].entry.due_date)

		time_ci_prev, _ := time.Parse(layout, assignments[i-1].entry.due_date)
		time_ci, _ := time.Parse(layout, assignments[i].entry.submitted_date)

		durations[i] = (float64(time_di.UnixNano()) - float64(time_di_prev.UnixNano())) / 1e9
		grade_proportions[i] = float64(assignments[i].entry.points)
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
		time_i, _ := time.Parse(layout, assignments[i].entry.due_date)
		time_j, _ := time.Parse(layout, assignments[j].entry.due_date)
		today_duration_i := today_time - (float64(time_i.UnixNano()) / 1e9)
		today_duration_j := today_time - (float64(time_j.UnixNano()) / 1e9)
		grade_proportion_i := assignments[i].entry.points
		grade_proportion_j := assignments[j].entry.points
		expected_time_i := linear_reg_model(assignments[i].duration, float64(grade_proportion_i))
		expected_time_j := linear_reg_model(assignments[j].duration, float64(grade_proportion_j))
		return (model.UrgencyDetermination(today_duration_i, expected_time_i, float64(grade_proportion_i)) > model.UrgencyDetermination(today_duration_j, expected_time_j, float64(grade_proportion_j)))
	})

	return unsubmitted_assignments
}

func assignments_to_exits(assignments []assignment) (exits []exit) {
	n := len(assignments)
	exit_slice := make([]exit, n)
	cash := 100
	for i := 0; i < n; i++ {
		exit_slice[i].course_name = assignments[i].entry.course_name
		exit_slice[i].assign_name = assignments[i].entry.assign_name
		exit_slice[i].due_date = assignments[i].entry.due_date
		if i < 3 {
			exit_slice[i].money = cash
			cash = cash / 2
		} else if i == 3 {
			exit_slice[i].money = 10
		} else {
			exit_slice[i].money = 5
		}
	}
	return exit_slice

}

func main() {

	entries := []entry{
		{
			course_name:     "Math 101",
			assign_name:     "Homework 1",
			due_date:        "2025-04-10T23:59:00Z",
			submitted_date:  "2025-04-09T20:00:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-01T12:00:00Z",
			category_weight: 0.1,
			points:          100,
		},
		{
			course_name:     "Physics 202",
			assign_name:     "Lab Report 1",
			due_date:        "2025-04-12T23:59:00Z",
			submitted_date:  "2025-04-12T22:30:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-03T10:00:00Z",
			category_weight: 0.15,
			points:          50,
		},
		{
			course_name:     "CompSci 303",
			assign_name:     "Project Proposal",
			due_date:        "2025-04-15T23:59:00Z",
			submitted_date:  "",
			is_submitted:    false,
			date_posted:     "2025-04-05T15:00:00Z",
			category_weight: 0.2,
			points:          25,
		},
		{
			course_name:     "Math 101",
			assign_name:     "Homework 2",
			due_date:        "2025-04-18T23:59:00Z",
			submitted_date:  "2025-04-17T19:00:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-10T08:00:00Z",
			category_weight: 0.1,
			points:          100,
		},
		{
			course_name:     "Physics 202",
			assign_name:     "Problem Set",
			due_date:        "2025-04-20T23:59:00Z",
			submitted_date:  "",
			is_submitted:    false,
			date_posted:     "2025-04-12T11:00:00Z",
			category_weight: 0.1,
			points:          40,
		},
		{
			course_name:     "CompSci 303",
			assign_name:     "Midterm Report",
			due_date:        "2025-04-22T23:59:00Z",
			submitted_date:  "2025-04-21T23:30:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-14T13:00:00Z",
			category_weight: 0.3,
			points:          60,
		},
		{
			course_name:     "Math 101",
			assign_name:     "Quiz 1",
			due_date:        "2025-04-24T10:00:00Z",
			submitted_date:  "2025-04-24T09:50:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-18T07:00:00Z",
			category_weight: 0.05,
			points:          20,
		},
		{
			course_name:     "Physics 202",
			assign_name:     "Lab Report 2",
			due_date:        "2025-04-26T23:59:00Z",
			submitted_date:  "",
			is_submitted:    false,
			date_posted:     "2025-04-19T09:00:00Z",
			category_weight: 0.15,
			points:          50,
		},
		{
			course_name:     "CompSci 303",
			assign_name:     "Code Review",
			due_date:        "2025-04-28T23:59:00Z",
			submitted_date:  "2025-04-28T18:00:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-20T15:30:00Z",
			category_weight: 0.1,
			points:          30,
		},
		{
			course_name:     "Math 101",
			assign_name:     "Homework 3",
			due_date:        "2025-05-01T23:59:00Z",
			submitted_date:  "2025-04-30T20:00:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-22T10:00:00Z",
			category_weight: 0.1,
			points:          100,
		},
		{
			course_name:     "Physics 202",
			assign_name:     "Final Prep",
			due_date:        "2025-05-04T23:59:00Z",
			submitted_date:  "",
			is_submitted:    false,
			date_posted:     "2025-04-26T11:00:00Z",
			category_weight: 0.2,
			points:          70,
		},
		{
			course_name:     "CompSci 303",
			assign_name:     "Final Project",
			due_date:        "2025-05-08T23:59:00Z",
			submitted_date:  "2025-05-08T22:00:00Z",
			is_submitted:    true,
			date_posted:     "2025-04-28T13:30:00Z",
			category_weight: 0.4,
			points:          100,
		},
	}

	fmt.Println(entry_processor(entries))
	fmt.Println(assignments_to_exits(urgency_sort(entry_processor(entries))))
}
