package model

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

func LinearRegressionModel(x1 []float64, x2 []float64, y1 []float64) func(float64, float64) float64 {
	n := len(x1)

	// Handle empty input
	if n == 0 {
		fmt.Println("hello")
		return func(time_duration float64, grade_proportion float64) float64 {
			return time_duration
		}
	}

	// Define the number of columns in the design matrix (X)
	rows := 3 // bias (ones), x1, x2

	// Create a slice for the ones column (bias term)
	ones := make([]float64, n)
	for i := range ones {
		ones[i] = 1.0
	}

	// Create the data slice to store the design matrix X (with bias term)
	data := make([]float64, 0, n*3)
	for i := 0; i < n; i++ {
		// Append (bias term, x1, x2) for each row
		data = append(data, ones[i], x1[i], x2[i])
	}

	// Create the X matrix (design matrix)
	X := mat.NewDense(n, rows, data)

	// Step 1: Compute X^T (transpose of X)
	var XT mat.Dense
	XT.CloneFrom(X.T())

	// Step 2: Compute X^T * X
	var XTX mat.Dense
	XTX.Mul(&XT, X)

	// Step 3: Compute the inverse of (X^T * X)
	var XTXInv mat.Dense
	if err := XTXInv.Inverse(&XTX); err != nil {
		fmt.Println("Matrix inversion failed")
		return func(time_duration float64, grade_proportion float64) float64 {
			return time_duration
		}
	}

	// Step 4: Compute X^T * y
	y := mat.NewVecDense(len(y1), y1)
	var XTy mat.VecDense
	XTy.MulVec(&XT, y)

	// Step 5: Compute (X^T * X)^(-1) * X^T * y to get the regression coefficients (beta)
	var beta mat.VecDense
	beta.MulVec(&XTXInv, &XTy)

	// Return a function that computes the prediction given the coefficients
	return func(time_duration float64, grade_proportion float64) float64 {
		// y = beta_0 + beta_1 * x1 + beta_2 * x2
		return beta.AtVec(0) + time_duration*beta.AtVec(1) + grade_proportion*beta.AtVec(2)
	}
}

func UrgencyDetermination(due_time float64, today_time float64, expected_time float64, grade_proportion float64) float64 {
	tuning_factor := 1.0
	return math.Pow(math.E, tuning_factor*(today_time+expected_time-due_time)) * grade_proportion

}

// func main() {
// 	// Example data
// 	x1 := []float64{1, 2, 3, 4}
// 	x2 := []float64{1, 0, 3, 5}
// 	y1 := []float64{1.038, 4.905, 6.113, 1.357}

// 	// Create the linear regression model
// 	model := LinearRegressionModel(x1, x2, y1)

// 	// Use the model to predict for new inputs
// 	fmt.Printf("Prediction for (1.5, 1.0): %v\n", model(1.5, 1.0))
// }
