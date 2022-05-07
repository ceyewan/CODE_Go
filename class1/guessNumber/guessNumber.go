package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func main() {
	maxNum := 100
	// Unix 方法获取时间戳，秒数; UnixNano 方法获取时间戳，纳秒数
	rand.Seed(time.Now().Unix())      // 随机数种子
	secretNumber := rand.Intn(maxNum) // Intn 返回 [0, n)
	fmt.Println("Please input your guess")
	// reader := bufio.NewReader(os.Stdin) // 创建了一个输入流
Loop:
	for {
		var input string
		_, err := fmt.Scanf("%s", &input)
		// input, err := reader.ReadString('\n') // 读取直到 '\n'
		if err != nil {
			fmt.Println("Reading error")
			continue
		}
		// input = strings.TrimSuffix(input, "\n") // 删除字符串 "\n"
		// input = input[:len(input)-1]      // 因为我们知道换行符只出现在最后
		guess, err := strconv.Atoi(input) // 要求 input 为纯数字
		if err != nil {
			fmt.Println("Invalid input")
			continue
		}
		fmt.Println("Your guess is:", guess)
		switch {
		case guess > secretNumber:
			fmt.Println("Bigger")
		case guess < secretNumber:
			fmt.Println("Smaller")
		default:
			fmt.Println("You are right!")
			break Loop // 跳出外层循环
		}
	}
}
