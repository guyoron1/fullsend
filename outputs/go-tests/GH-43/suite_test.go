package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGH43(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GH-43 Harness-First Agent Discovery Suite")
}
