package db

import (
	"database/sql"
	"fmt"
	"slices"
	"strconv"
)

func (d DataBase) CalculateIP() ([]int, error) {
	var (
		IPsPool []int
		IPs     []string
	)

	qIProw, err := d.DataBase.Query("SELECT IP from registration_queue")
	if err != nil {
		return nil, fmt.Errorf("RegisterQueue: db: failed to query IPs from registration_queue: %w", err)
	}
	defer func(qIProw *sql.Rows) {
		err := qIProw.Close()
		if err != nil {
			fmt.Printf("RegisterQueue: failed to close DB rows: %v", err)
		}
	}(qIProw)
	for qIProw.Next() {
		var IP string
		err = qIProw.Scan(&IP)
		if err != nil {
			return nil, fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		IPs = append(IPs, IP)
	}

	uIProw, err := d.DataBase.Query("SELECT IP from users")
	if err != nil {
		return nil, fmt.Errorf("func RegisterQueue: db: failed to query IPs from users: %w", err)
	}
	defer func(uIProw *sql.Rows) {
		err := uIProw.Close()
		if err != nil {
			fmt.Printf("RegisterQueue: failed to close DB rows: %v", err)
		}
	}(uIProw)
	for uIProw.Next() {
		var IP string
		err = uIProw.Scan(&IP)
		if err != nil {
			return nil, fmt.Errorf("func RegisterQueue: db: failed to get row value: %w", err)
		}
		IPs = append(IPs, IP)
	}
	for i := 130; i < 255; i++ {
		IPsPool = append(IPsPool, i)
	}

	IPsOctet := slices.DeleteFunc(IPsPool, func(n int) bool {
		return slices.Contains(IPs, strconv.Itoa(n))
	})
	return IPsOctet, nil
}
