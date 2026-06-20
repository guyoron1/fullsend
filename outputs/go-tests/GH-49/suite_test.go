package tests

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAgentSlugDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Slug Discovery Suite — GH-49")
}
