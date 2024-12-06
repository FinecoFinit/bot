package concierge

import "slices"

func CalculateIP(uPool []int, quPool []int) []int {
	var IPsPool []int
	combPool := append(uPool, quPool...)
	for i := 130; i < 255; i++ {
		IPsPool = append(IPsPool, i)
	}
	freeIPs := slices.DeleteFunc(IPsPool, func(n int) bool {
		return slices.Contains(combPool, n)
	})
	return freeIPs
}
