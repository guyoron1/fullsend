package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnsureProvider(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EnsureProvider Suite [GH-28]")
}
