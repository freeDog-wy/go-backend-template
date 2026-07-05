package crypto

import (
	"fmt"
	"testing"
)

func TestPasswordHasher(t *testing.T) {
	phasher := NewBcryptHasher(0)

	pass := "123456"
	passHashed, err := phasher.Hash(pass)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	fmt.Println("pass: ", passHashed)

	if !phasher.Verify(pass, passHashed) {
		t.Errorf("Password verification failed")
	}
}
