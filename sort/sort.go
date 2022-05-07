package main

import (
	"fmt"
	"math/rand"
	"sort"
)

func BubbleSort(nums []int) {
	n := len(nums)
	swap := false
	for i := n - 1; i > 0; i-- {
		for j := 0; j < i; j++ {
			if nums[j] > nums[j+1] {
				nums[j], nums[j+1] = nums[j+1], nums[j]
				swap = true
			}
		}
		if !swap {
			break
		}
	}
}

func SelectSort(nums []int) {
	n := len(nums)
	for i := 0; i < n/2; i++ {
		min, minindex := nums[i], i
		max, maxindex := nums[i], i
		for j := i + 1; j < n-i; j++ {
			if nums[j] < min {
				min, minindex = nums[j], j
			}
			if nums[j] > max {
				max, maxindex = nums[j], j
			}
		}
		nums[i], nums[minindex] = nums[minindex], nums[i]
		nums[n-1-i], nums[maxindex] = nums[maxindex], nums[n-i-1]
	}
}

func InsertSort(nums []int) {
	n := len(nums)
	for i := 1; i < n; i++ {
		for j := i - 1; j >= 0; j-- {
			if nums[j] > nums[j+1] {
				nums[j], nums[j+1] = nums[j+1], nums[j]
			} else {
				break
			}
		}
	}
}

func ShellSort(nums []int) {
	n := len(nums)
	for step := n / 2; step >= 1; step-- {
		// 步长为 step 的插入排序
		for i := step; i < n; i += step {
			for j := i - step; j >= 0; j -= step {
				if nums[j+step] < nums[j] {
					nums[j], nums[j+step] = nums[j+step], nums[j]
				} else {
					break
				}
			}
		}
	}
}

func MergeSort(nums []int) {
	if len(nums) == 1 {
		return
	}
	nums1 := nums[:len(nums)/2]
	nums2 := nums[len(nums)/2:]
	MergeSort(nums1)
	MergeSort(nums2)
	temp := make([]int, len(nums))
	index := 0
	for i, j := 0, 0; i < len(nums1) || j < len(nums2); index++ {
		if i == len(nums1) {
			temp[index] = nums2[j]
			j++
		} else if j == len(nums2) {
			temp[index] = nums1[i]
			i++
		} else if nums1[i] < nums2[j] {
			temp[index] = nums1[i]
			i++
		} else {
			temp[index] = nums2[j]
			j++
		}
	}
	copy(nums, temp)
}

type Heap struct {
	Size  int
	Array []int
}

func NewHeap(nums []int) *Heap {
	h := Heap{
		Size:  0,
		Array: nums,
	}
	return &h
}

func (h *Heap) Push(x int) {
	i := h.Size
	h.Array[i] = x
	for i > 0 {
		parent := (i - 1) / 2
		if h.Array[i] <= h.Array[parent] {
			break
		}
		h.Array[i], h.Array[parent] = h.Array[parent], h.Array[i]
		i = parent
	}
	h.Size++
}

func (h *Heap) Pop() int {
	if h.Size == 0 {
		return -1
	}
	ret := h.Array[0]
	h.Size--
	h.Array[0] = h.Array[h.Size]
	i := 0
	for {
		a, b := 2*i+1, 2*(i+1)
		if a >= h.Size {
			break
		}
		if b < h.Size && h.Array[b] > h.Array[a] {
			a = b
		}
		if h.Array[i] >= h.Array[a] {
			break
		}
		h.Array[i], h.Array[a] = h.Array[a], h.Array[i]
		i = a
	}
	return ret
}

func HeapSort(nums []int) {
	h := NewHeap(nums)
	for _, v := range nums {
		h.Push(v)
	}
	for i := len(nums) - 1; i >= 0; i-- {
		nums[i] = h.Pop()
	}
}

func partition(nums []int, start, end int) int {
	temp := nums[start]
	i, j := start, end
	for i < j {
		for i < j && nums[j] >= temp {
			j--
		}
		if i < j {
			nums[i] = nums[j]
			i++
		}
		for i < j && nums[i] < temp {
			i++
		}
		if i < j {
			nums[j] = nums[i]
		}
	}
	nums[i] = temp
	return i
}
func QuickSort(nums []int, start, end int) {
	if start >= end {
		return
	}
	i := partition(nums, start, end)
	QuickSort(nums, start, i-1)
	QuickSort(nums, i+1, end)
}

type IntSlice []int

func (m IntSlice) Len() int {
	return len(m)
}
func (m IntSlice) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}
func (m IntSlice) Less(i, j int) bool {
	return m[i] < m[j]
}

func main() {
	nums := make([]int, 1000)
	for i := 0; i < 1000; i++ {
		nums[i] = int(rand.Int31()) % 1000
	}
	BubbleSort(nums)
	SelectSort(nums)
	InsertSort(nums)
	ShellSort(nums)
	MergeSort(nums)
	HeapSort(nums)
	QuickSort(nums, 0, len(nums)-1)
	sort.Sort(IntSlice(nums))
	sort.Ints(nums)
	sort.Slice(nums, func(i, j int) bool {
		return nums[i] >= nums[j]
	})
	fmt.Println(nums)
}
