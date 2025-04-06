package model

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
)

// LinearRegressionModel performs linear regression using Singular Value Decomposition (SVD)
// func LinearRegressionModel(x1 []float64, x2 []float64, y1 []float64) func(float64, float64) float64 {
// 	n := len(x1)

// 	// Handle empty input
// 	if n == 0 {
// 		fmt.Println("No data to train on")
// 		return func(time_duration float64, grade_proportion float64) float64 {
// 			return 0.0
// 		}
// 	}

// 	// Number of features (including bias term)
// 	rows := 3 // for bias (ones), x1, and x2

// 	// Create the design matrix X (with bias term)
// 	data := make([]float64, 0, n*3)
// 	for i := 0; i < n; i++ {
// 		// Append (bias term, x1, x2) for each row
// 		data = append(data, 1.0, x1[i], x2[i])
// 	}

// 	// Create the X matrix (design matrix) as a dense matrix
// 	X := mat.NewDense(n, rows, data)

// 	// Create the Y vector (response variable)
// 	y := mat.NewVecDense(len(y1), y1)

// 	// Step 1: Perform Singular Value Decomposition (SVD) on X
// 	var svd mat.SVD
// 	ok := svd.Factorize(X, mat.SVDThin)
// 	if !ok {
// 		log.Fatal("SVD factorization failed")
// 	}

// 	// Step 2: Compute the solution (beta) using the SVD: beta = V * Sigma^-1 * U^T * y
// 	// First, calculate U^T * y
// 	var UTransY mat.VecDense
// 	UTransY.MulVec(svd.UTo(), y)

// 	// Create the inverse of the singular values (Sigma^-1)
// 	sigmaInv := make([]float64, len(svd.Values))
// 	for i := 0; i < len(svd.Values); i++ {
// 		if svd.Values[i] > 1e-10 { // Avoid division by zero
// 			sigmaInv[i] = 1.0 / svd.Values[i]
// 		} else {
// 			sigmaInv[i] = 0
// 		}
// 	}

// 	// Now, multiply U^T * y by the inverse of the singular values
// 	for i := 0; i < len(svd.Values); i++ {
// 		UTransY.SetVec(i, UTransY.AtVec(i)*sigmaInv[i])
// 	}

// 	// Multiply by V to get the final beta coefficients
// 	var beta mat.VecDense
// 	beta.MulVec(svd.Vt, &UTransY)

// 	// Return a function that computes the prediction given the coefficients
// 	return func(time_duration float64, grade_proportion float64) float64 {
// 		// y = beta_0 + beta_1 * x1 + beta_2 * x2
// 		return beta.At(0) + time_duration*beta.At(1) + grade_proportion*beta.At(2)
// 	}
// }

func LinearRegressionModel(x1 []float64, x2 []float64, y1 []float64) func(float64, float64) float64 {
	n := len(x1)
	if n == 0 {
		fmt.Println("No data to train on")
		return func(time_duration float64, grade_proportion float64) float64 {
			return 0.0
		}
	}
	fmt.Println(x1)
	fmt.Println(x2)
	rows := 3
	data := make([]float64, 0, n*rows)
	for i := 0; i < n; i++ {
		data = append(data, 1.0, x1[i], x2[i])
	}
	X := mat.NewDense(n, rows, data)
	y := mat.NewVecDense(len(y1), y1)

	// Compute XᵗX
	var XT mat.Dense
	XT.CloneFrom(X.T())
	var XTX mat.Dense
	XTX.Mul(&XT, X)

	// Regularize: Add λI
	lambda := 1e-8
	for i := 0; i < rows; i++ {
		XTX.Set(i, i, XTX.At(i, i)+lambda)
	}

	// Compute Xᵗy
	var XTy mat.VecDense
	XTy.MulVec(&XT, y)

	// Solve (XᵗX + λI) β = Xᵗy
	var beta mat.VecDense
	err := beta.SolveVec(&XTX, &XTy)

	fmt.Println(X)
	if err != nil {
		fmt.Println("Solve failed:", err)
		return func(time_duration float64, grade_proportion float64) float64 {
			return 0.0
		}
	}

	// Return predictor
	return func(time_duration float64, grade_proportion float64) float64 {
		return beta.AtVec(0) + time_duration*beta.AtVec(1) + grade_proportion*beta.AtVec(2)
	}
}

func UrgencyDetermination(late_time float64, expected_time float64, grade_proportion float64) float64 {
	tuning_factor := 0.01
	return grade_proportion * tuning_factor / math.Abs((late_time + expected_time))
	//return math.Pow(math.E, tuning_factor*(late_time+expected_time)) * grade_proportion / (late_time+expected_time)

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
