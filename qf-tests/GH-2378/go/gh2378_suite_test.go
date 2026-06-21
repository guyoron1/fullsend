//go:build e2e

package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGH2378(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "GH-2378 Agent Status Comment Suite")
}
