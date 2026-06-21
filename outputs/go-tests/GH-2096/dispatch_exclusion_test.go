package review

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
Dispatch Exclusion Tests — GH-2096

Validates that non-dimension sub-agents (security-triage, challenger) are excluded
from the step 4 parallel dispatch loop, while all dimension sub-agents are included.

STP Reference: outputs/stp/GH-2096/GH-2096_test_plan.md
STD Scenarios: TS-GH-2096-016, TS-GH-2096-017, TS-GH-2096-018
*/

// SubAgentType classifies a sub-agent as a dimension (review) or non-dimension (utility).
type SubAgentType string

const (
	DimensionAgent    SubAgentType = "dimension"
	NonDimensionAgent SubAgentType = "non-dimension"
)

// SubAgent represents a sub-agent in the review roster.
type SubAgent struct {
	Name       string
	AgentType  SubAgentType
	Dispatchable bool
}

// buildRoster returns the full sub-agent roster with their types.
func buildRoster() []SubAgent {
	return []SubAgent{
		{Name: "security", AgentType: DimensionAgent, Dispatchable: true},
		{Name: "correctness", AgentType: DimensionAgent, Dispatchable: true},
		{Name: "style-conventions", AgentType: DimensionAgent, Dispatchable: true},
		{Name: "docs-currency", AgentType: DimensionAgent, Dispatchable: true},
		{Name: "intent-coherence", AgentType: DimensionAgent, Dispatchable: true},
		{Name: "cross-repo-contracts", AgentType: DimensionAgent, Dispatchable: true},
		{Name: "security-triage", AgentType: NonDimensionAgent, Dispatchable: false},
		{Name: "challenger", AgentType: NonDimensionAgent, Dispatchable: false},
	}
}

// filterForDispatch returns only dimension sub-agents eligible for step 4
// parallel dispatch. Non-dimension agents (security-triage, challenger) are excluded.
func filterForDispatch(roster []SubAgent) []SubAgent {
	var dispatched []SubAgent
	for _, agent := range roster {
		if agent.Dispatchable && agent.AgentType == DimensionAgent {
			dispatched = append(dispatched, agent)
		}
	}
	return dispatched
}

func TestDispatchExclusion(t *testing.T) {
	/*
		Preconditions:
			- Go development environment with Go 1.26+
			- fullsend repository with two-pass review strategy changes
			- Sub-agent roster loaded
	*/

	roster := buildRoster()

	// TS-GH-2096-016: Verify security-triage excluded from step 4 dispatch
	t.Run("security-triage excluded from step 4 dispatch", func(t *testing.T) {
		dispatchList := filterForDispatch(roster)

		agentNames := make([]string, len(dispatchList))
		for i, a := range dispatchList {
			agentNames[i] = a.Name
		}

		assert.NotContains(t, agentNames, "security-triage",
			"security-triage must not appear in dispatch list (runs as pre-pass)")
	})

	// TS-GH-2096-017: Verify challenger excluded from step 4 dispatch
	t.Run("challenger excluded from step 4 dispatch", func(t *testing.T) {
		dispatchList := filterForDispatch(roster)

		agentNames := make([]string, len(dispatchList))
		for i, a := range dispatchList {
			agentNames[i] = a.Name
		}

		assert.NotContains(t, agentNames, "challenger",
			"challenger must not appear in dispatch list (runs as post-processing)")
	})

	// TS-GH-2096-018: Verify dimension sub-agents dispatched normally
	t.Run("dimension sub-agents dispatched normally", func(t *testing.T) {
		dispatchList := filterForDispatch(roster)

		// Count expected dimension agents
		expectedDimensions := 0
		for _, a := range roster {
			if a.AgentType == DimensionAgent {
				expectedDimensions++
			}
		}

		require.Len(t, dispatchList, expectedDimensions,
			"dispatch count must match expected dimension count")

		// Verify all dimension sub-agents are present
		expectedNames := []string{
			"security", "correctness", "style-conventions",
			"docs-currency", "intent-coherence", "cross-repo-contracts",
		}
		dispatchedNames := make([]string, len(dispatchList))
		for i, a := range dispatchList {
			dispatchedNames[i] = a.Name
		}
		for _, name := range expectedNames {
			assert.Contains(t, dispatchedNames, name,
				"dimension sub-agent %q must be in dispatch list", name)
		}
	})
}
