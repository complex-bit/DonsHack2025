package main

type assignment struct {
	course_name     string
	assign_name     string
	due_date        string
	submitted_date  string
	category_weight float64
	points          int
}

type entry struct {
	course_name string
	assign_name string
	due_date    string
	money       int
}

func data_chugjug(assignments []assignment) (entries []entry) {

}
