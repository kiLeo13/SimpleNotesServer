package utils

const CNPJLength = 14

func IsCNPJValid(cnpj string) bool {
	if len(cnpj) != CNPJLength {
		return false
	}

	if !IsOnlyNumbers(cnpj) {
		return false
	}

	// Reject known invalid patterns that trick the math algorithm
	if hasAllSameDigits(cnpj) {
		return false
	}
	return validateCNPJDigits(cnpj)
}

func hasAllSameDigits(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func validateCNPJDigits(cnpj string) bool {
	// RFB weights for the first verifying digit
	weights1 := []int{5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}
	// RFB weights for the second verifying digit
	weights2 := []int{6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2}

	digit1 := calculateCNPJDigit(cnpj[:12], weights1)
	digit2 := calculateCNPJDigit(cnpj[:13], weights2)

	actualDigit1 := int(cnpj[12] - '0')
	actualDigit2 := int(cnpj[13] - '0')

	return digit1 == actualDigit1 && digit2 == actualDigit2
}

func calculateCNPJDigit(base string, weights []int) int {
	sum := 0
	for i, weight := range weights {
		// Convert ASCII character to integer ('5' -> 5)
		digit := int(base[i] - '0')
		sum += digit * weight
	}

	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}
