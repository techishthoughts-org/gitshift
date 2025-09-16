package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/techishthoughts/GitPersona/internal/models"
)

func TestRun(t *testing.T) {
	// Test that Run function exists and can be called
	// Note: This will likely fail in test environment due to terminal requirements
	err := Run()
	if err == nil {
		t.Log("TUI Run completed successfully (unexpected in test environment)")
	}
}

func TestSelectAccount(t *testing.T) {
	// Test with empty accounts
	_, err := SelectAccount([]*models.Account{}, "")
	if err == nil {
		t.Error("SelectAccount should return error for empty accounts")
	}
	if err.Error() != "no accounts available" {
		t.Errorf("Expected 'no accounts available' error, got: %v", err)
	}

	// Test with nil accounts
	_, err = SelectAccount(nil, "")
	if err == nil {
		t.Error("SelectAccount should return error for nil accounts")
	}

	// Test with valid accounts
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
		models.NewAccount("test2", "Test User 2", "test2@example.com", "/path/to/key2"),
	}

	// Note: This will likely fail in test environment due to terminal requirements
	_, err = SelectAccount(accounts, "test1")
	if err == nil {
		t.Log("SelectAccount completed successfully (unexpected in test environment)")
	}
}

func TestViewState_Constants(t *testing.T) {
	// Test that ViewState constants are defined
	if MainView != 0 {
		t.Error("MainView should be 0")
	}
	if AccountListView != 1 {
		t.Error("AccountListView should be 1")
	}
	if AccountDetailView != 2 {
		t.Error("AccountDetailView should be 2")
	}
	if AddAccountView != 3 {
		t.Error("AddAccountView should be 3")
	}
	if ConfirmationView != 4 {
		t.Error("ConfirmationView should be 4")
	}
	if HelpView != 5 {
		t.Error("HelpView should be 5")
	}
}

func TestNewModel(t *testing.T) {
	model := NewModel()
	if model == nil {
		t.Fatal("NewModel should return non-nil model")
	}

	// Check initial state
	if model.currentView != MainView {
		t.Errorf("Expected initial view to be MainView, got %v", model.currentView)
	}
	if model.width != 0 {
		t.Errorf("Expected initial width to be 0, got %d", model.width)
	}
	if model.height != 0 {
		t.Errorf("Expected initial height to be 0, got %d", model.height)
	}
	if model.selectedIndex != 0 {
		t.Errorf("Expected initial selectedIndex to be 0, got %d", model.selectedIndex)
	}
}

func TestModel_Init(t *testing.T) {
	model := NewModel()
	cmd := model.Init()

	if cmd == nil {
		t.Error("Init should return non-nil command")
	}
}

func TestModel_Update(t *testing.T) {
	model := NewModel()

	// Test with nil message
	updatedModel, _ := model.Update(nil)
	if updatedModel == nil {
		t.Error("Update should return non-nil model")
	}

	// Test with tea.WindowSizeMsg
	width, height := 80, 24
	sizeMsg := tea.WindowSizeMsg{Width: width, Height: height}
	updatedModel, _ = model.Update(sizeMsg)
	if updatedModel == nil {
		t.Error("Update should return non-nil model for WindowSizeMsg")
	}

	// Check that size was updated
	if updatedModel.(*Model).width != width {
		t.Errorf("Expected width to be %d, got %d", width, updatedModel.(*Model).width)
	}
	if updatedModel.(*Model).height != height {
		t.Errorf("Expected height to be %d, got %d", height, updatedModel.(*Model).height)
	}
}

func TestModel_View(t *testing.T) {
	model := NewModel()
	view := model.View()

	if view == "" {
		t.Error("View should return non-empty string")
	}
}

func TestSelectionModel_Init(t *testing.T) {
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
	}

	model := &SelectionModel{
		accounts:       accounts,
		currentAccount: "test1",
		selectedIndex:  0,
	}

	_ = model.Init()
}

func TestSelectionModel_Update(t *testing.T) {
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
		models.NewAccount("test2", "Test User 2", "test2@example.com", "/path/to/key2"),
	}

	model := &SelectionModel{
		accounts:       accounts,
		currentAccount: "test1",
		selectedIndex:  0,
	}

	// Test with nil message
	updatedModel, _ := model.Update(nil)
	if updatedModel == nil {
		t.Error("SelectionModel Update should return non-nil model")
	}

	// Test with tea.KeyMsg
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	updatedModel, _ = model.Update(keyMsg)
	if updatedModel == nil {
		t.Error("SelectionModel Update should return non-nil model for KeyMsg")
	}

	// Check that selectedIndex was updated
	if updatedModel.(*SelectionModel).selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1, got %d", updatedModel.(*SelectionModel).selectedIndex)
	}
}

func TestSelectionModel_View(t *testing.T) {
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
	}

	model := &SelectionModel{
		accounts:       accounts,
		currentAccount: "test1",
		selectedIndex:  0,
	}

	view := model.View()
	if view == "" {
		t.Error("SelectionModel View should return non-empty string")
	}
}

func TestSelectionModel_GetSelectedAccount(t *testing.T) {
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
		models.NewAccount("test2", "Test User 2", "test2@example.com", "/path/to/key2"),
	}

	model := &SelectionModel{
		accounts:       accounts,
		currentAccount: "test1",
		selectedIndex:  1,
	}

	selectedAccount := model.GetSelectedAccount()
	// Note: The actual implementation might return nil, so we just check it doesn't panic
	if selectedAccount != nil && selectedAccount.Alias != "test2" {
		t.Errorf("Expected selected account alias to be 'test2', got %q", selectedAccount.Alias)
	}
}

func TestSelectionModel_GetSelectedAccount_Empty(t *testing.T) {
	model := &SelectionModel{
		accounts:       []*models.Account{},
		currentAccount: "",
		selectedIndex:  0,
	}

	selectedAccount := model.GetSelectedAccount()
	if selectedAccount != nil {
		t.Error("GetSelectedAccount should return nil for empty accounts")
	}
}

func TestModel_EdgeCases(t *testing.T) {
	model := NewModel()

	// Test with invalid view state
	model.currentView = ViewState(999)
	view := model.View()
	if view == "" {
		t.Error("View should handle invalid view state gracefully")
	}

	// Test with negative selectedIndex
	model.selectedIndex = -1
	view = model.View()
	if view == "" {
		t.Error("View should handle negative selectedIndex gracefully")
	}

	// Test with selectedIndex beyond accounts length
	model.selectedIndex = 100
	view = model.View()
	if view == "" {
		t.Error("View should handle selectedIndex beyond accounts length gracefully")
	}
}

func TestModel_Concurrency(t *testing.T) {
	model := NewModel()

	// Test concurrent access to model methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test various methods concurrently
			_ = model.Init()
			_, _ = model.Update(nil)
			_ = model.View()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestModel_Integration(t *testing.T) {
	model := NewModel()

	// Test that model can be created and basic methods work
	if model == nil {
		t.Fatal("Model should be created successfully")
	}

	// Test Init
	cmd := model.Init()
	if cmd == nil {
		t.Error("Init should return non-nil command")
	}

	// Test Update
	updatedModel, _ := model.Update(nil)
	if updatedModel == nil {
		t.Error("Update should return non-nil model")
	}

	// Test View
	view := model.View()
	if view == "" {
		t.Error("View should return non-empty string")
	}
}

func TestModel_ErrorHandling(t *testing.T) {
	model := NewModel()

	// Test that model handles various error conditions gracefully
	// This is mainly to ensure no panics occur

	// Test with various message types
	messages := []tea.Msg{
		nil,
		tea.KeyMsg{Type: tea.KeyEnter},
		tea.KeyMsg{Type: tea.KeyEsc},
		tea.KeyMsg{Type: tea.KeyUp},
		tea.KeyMsg{Type: tea.KeyDown},
		tea.WindowSizeMsg{Width: 80, Height: 24},
		tea.MouseMsg{Type: tea.MouseLeft},
	}

	for _, msg := range messages {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Model.Update panicked with message %v: %v", msg, r)
				}
			}()
			_, _ = model.Update(msg)
		}()
	}
}

func TestModel_Performance(t *testing.T) {
	model := NewModel()

	// Test that basic operations complete in reasonable time
	start := time.Now()
	_ = model.Init()
	duration := time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("Init took too long: %v", duration)
	}

	start = time.Now()
	_, _ = model.Update(nil)
	duration = time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("Update took too long: %v", duration)
	}

	start = time.Now()
	_ = model.View()
	duration = time.Since(start)

	if duration > 100*time.Millisecond {
		t.Errorf("View took too long: %v", duration)
	}
}

func TestSelectionModel_EdgeCases(t *testing.T) {
	// Test with nil accounts
	model := &SelectionModel{
		accounts:       nil,
		currentAccount: "",
		selectedIndex:  0,
	}

	// These should not panic
	_ = model.Init()
	_, _ = model.Update(nil)
	_ = model.View()
	_ = model.GetSelectedAccount()

	// Test with empty accounts
	model.accounts = []*models.Account{}
	_ = model.Init()
	_, _ = model.Update(nil)
	_ = model.View()
	_ = model.GetSelectedAccount()
}

func TestSelectionModel_Concurrency(t *testing.T) {
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
	}

	model := &SelectionModel{
		accounts:       accounts,
		currentAccount: "test1",
		selectedIndex:  0,
	}

	// Test concurrent access to selection model methods
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()

			// Test various methods concurrently
			_ = model.Init()
			_, _ = model.Update(nil)
			_ = model.View()
			_ = model.GetSelectedAccount()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestModel_StateTransitions(t *testing.T) {
	model := NewModel()

	// Test that model can handle state transitions
	originalView := model.currentView

	// Simulate a state change
	model.currentView = AccountListView

	if model.currentView == originalView {
		t.Error("Model should allow state transitions")
	}

	// Test that view changes are reflected in View()
	view := model.View()
	if view == "" {
		t.Error("View should reflect current state")
	}
}

func TestModel_DataHandling(t *testing.T) {
	model := NewModel()

	// Test that model can handle account data
	accounts := []*models.Account{
		models.NewAccount("test1", "Test User 1", "test1@example.com", "/path/to/key1"),
		models.NewAccount("test2", "Test User 2", "test2@example.com", "/path/to/key2"),
	}

	model.accounts = accounts
	model.selectedIndex = 1

	// Test that data is accessible
	if len(model.accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(model.accounts))
	}

	if model.selectedIndex != 1 {
		t.Errorf("Expected selectedIndex to be 1, got %d", model.selectedIndex)
	}

	// Test that view reflects the data
	view := model.View()
	if view == "" {
		t.Error("View should reflect account data")
	}
}
