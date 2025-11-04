package utils

import (
	"strconv"
	"strings"
)

// FormatPrecisionString formats a string number into a decimal string with the specified precision
func FormatPrecisionString(number string, precision int) string {
	if precision <= 0 {
		return number
	}

	// Removing a possible point in the input line
	number = strings.ReplaceAll(number, ".", "")

	// If the number is less than 1, add zeros
	for len(number) <= precision {
		number = "0" + number
	}

	// Insert a point
	decimalIndex := len(number) - precision
	formattedNumber := number[:decimalIndex] + "." + number[decimalIndex:]

	return formattedNumber
}

// FormatPrecisionNumber formats an integer number (as int64) into a decimal string with the specified precision
func FormatPrecisionNumber(number int64, precision int) string {
	return FormatPrecisionString(strconv.Itoa(int(number)), precision)
}
