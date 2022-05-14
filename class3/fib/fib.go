package fib

func Fib(n int) int {
	ans := []int{}
	for i := 0; i <= n; i++ {
		if i < 2 {
			ans = append(ans, i)
		} else {
			ans = append(ans, ans[i-1]+ans[i-2])
		}
	}
	return ans[n]
}
