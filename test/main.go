// Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.
// You may assume that each input would have exactly one solution, and you may not use the same element twice.
// You can return the answer in any order.
// Input: nums = [3,2,4], target = 6
// Output: [1,2]

package main

// func getIndexs(arr []int, target int) []int {
// 	var res = make([]int, 2)
// 	// for i := 0; i < len(arr); i++ {
// 	// 	for j := i + 1; j < len(arr); j++ {
// 	// 		if arr[i]+arr[j] == target {
// 	// 			res[0] = i
// 	// 			res[1] = i
// 	// 			return res
// 	// 		}

// 	// 	}
// 	// }
// 		objMap := make(map[int]int)

// 	return res
// }
// func main() {

// 	var arrr = []int{3, 2, 4}
// 	var target = 6
// 	fmt.Println(getIndexs(arrr, target))
// }

func factorial(n int)int {
	if n == 0 {
		return 1
	}
	return n * factorial(n-1)
}

func main() {
	var n = 5
	var arr = [7]int{}
	var sum = n * (n + 1) / 2
	var fact = factorial(n)
	var actualSum int
	for _, value := range arr {
		actualSum += value
	}
	var temp = actualSum - sum
	var ftemp = actualSum / fact

}
