// Copyright (c) 2022 Notch Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD 3-Clause License
// license that can be found in the LICENSE file.

package tun

import "runtime"

func TunName() string {
	switch runtime.GOOS {
	case "openbsd":
		return "tun"
	case "linux":
		return "ds0"
	case "darwin":
		return "utun100"
	case "windows":
		return "dotshake"
	}
	return "ds0"
}
