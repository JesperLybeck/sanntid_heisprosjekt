package execute

import "fmt"


func AppendToMatrix(Matrix [4][3]int, floor int, button int, value int) {
	Matrix[floor][button] = value
}

func PrintMatrix(Matrix [4][3]int) {
    for _, row := range Matrix {
        fmt.Println(row)
    }
}