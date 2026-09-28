// SPDX-FileCopyrightText: 2026 Sebastien Rousseau <sebastian.rousseau@gmail.com>
// SPDX-License-Identifier: Apache-2.0

package a2a_test

import (
	"fmt"
	"os"

	"satellion.com/passmcp-reporting/a2a"
)

// A gateway admitting an A2A agent reads the statement, verifies it offline,
// and gates on one verdict.
func Example() {
	b, err := os.ReadFile("testdata/statement.json")
	if err != nil {
		panic(err)
	}
	st, err := a2a.Parse(b)
	if err != nil {
		fmt.Println("refuse:", err)
		return
	}
	v, _ := st.VerdictFor("a2a.unauthenticated")
	fmt.Println(st.Covers("https://agent.example.com"), v.Status)
	// Output: true fail
}
