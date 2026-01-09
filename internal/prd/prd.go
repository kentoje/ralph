package prd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type UserStory struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	AcceptanceCriteria []string `json:"acceptanceCriteria"`
	Priority           int      `json:"priority"`
	Passes             bool     `json:"passes"`
	Notes              string   `json:"notes"`
}

type PRD struct {
	Project     string      `json:"project"`
	BranchName  string      `json:"branchName"`
	Description string      `json:"description"`
	UserStories []UserStory `json:"userStories"`
}

// Load reads the prd.json file from a project directory
func Load(projectDir string) (*PRD, error) {
	path := filepath.Join(projectDir, "prd.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var prd PRD
	if err := json.Unmarshal(data, &prd); err != nil {
		return nil, err
	}

	return &prd, nil
}

// Save writes the prd.json file to a project directory
func Save(projectDir string, prd *PRD) error {
	path := filepath.Join(projectDir, "prd.json")
	data, err := json.MarshalIndent(prd, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Exists checks if prd.json exists in a project directory
func Exists(projectDir string) bool {
	path := filepath.Join(projectDir, "prd.json")
	_, err := os.Stat(path)
	return err == nil
}

// CompletedCount returns the number of completed user stories
func (p *PRD) CompletedCount() int {
	count := 0
	for _, story := range p.UserStories {
		if story.Passes {
			count++
		}
	}
	return count
}

// TotalCount returns the total number of user stories
func (p *PRD) TotalCount() int {
	return len(p.UserStories)
}

// NextIncomplete returns the next incomplete user story
func (p *PRD) NextIncomplete() *UserStory {
	for i := range p.UserStories {
		if !p.UserStories[i].Passes {
			return &p.UserStories[i]
		}
	}
	return nil
}

// IsComplete returns true if all user stories pass
func (p *PRD) IsComplete() bool {
	for _, story := range p.UserStories {
		if !story.Passes {
			return false
		}
	}
	return len(p.UserStories) > 0
}
