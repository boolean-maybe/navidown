package navidown

import "testing"

type testState struct {
	value string
}

func TestNavigationHistory_NewHistory(t *testing.T) {
	h := NewNavigationHistory[testState](10)

	if h == nil {
		t.Fatal("NewNavigationHistory should not return nil")
	}

	if h.CanGoBack() {
		t.Error("New history should not have back entries")
	}

	if h.CanGoForward() {
		t.Error("New history should not have forward entries")
	}

	if h.BackStackSize() != 0 {
		t.Errorf("BackStackSize = %d, want 0", h.BackStackSize())
	}

	if h.ForwardStackSize() != 0 {
		t.Errorf("ForwardStackSize = %d, want 0", h.ForwardStackSize())
	}
}

func TestNavigationHistory_Push(t *testing.T) {
	h := NewNavigationHistory[testState](10)

	h.Push(testState{value: "page1"})

	if !h.CanGoBack() {
		t.Error("After push, should be able to go back")
	}

	if h.CanGoForward() {
		t.Error("After push, should not be able to go forward")
	}

	if h.BackStackSize() != 1 {
		t.Errorf("BackStackSize = %d, want 1", h.BackStackSize())
	}

	h.Push(testState{value: "page2"})

	if h.BackStackSize() != 2 {
		t.Errorf("BackStackSize = %d, want 2", h.BackStackSize())
	}
}

func TestNavigationHistory_PushClearsForward(t *testing.T) {
	h := NewNavigationHistory[testState](10)

	h.Push(testState{value: "page1"})
	h.PushToForward(testState{value: "page2"})

	if !h.CanGoForward() {
		t.Fatal("setup failed: should have forward history")
	}

	h.Push(testState{value: "page3"})

	if h.CanGoForward() {
		t.Error("Push should clear forward history")
	}

	if h.ForwardStackSize() != 0 {
		t.Errorf("ForwardStackSize = %d, want 0", h.ForwardStackSize())
	}
}

func TestNavigationHistory_Back(t *testing.T) {
	h := NewNavigationHistory[testState](10)

	h.Push(testState{value: "page1"})
	h.Push(testState{value: "page2"})

	state, ok := h.Back()
	if !ok {
		t.Fatal("Back should succeed when history exists")
	}

	if state.value != "page2" {
		t.Errorf("Back returned state.value = %q, want %q", state.value, "page2")
	}

	if h.BackStackSize() != 1 {
		t.Errorf("After back, BackStackSize = %d, want 1", h.BackStackSize())
	}

	state, ok = h.Back()
	if !ok {
		t.Fatal("Second back should succeed")
	}

	if state.value != "page1" {
		t.Errorf("Second back returned state.value = %q, want %q", state.value, "page1")
	}

	if h.BackStackSize() != 0 {
		t.Errorf("After second back, BackStackSize = %d, want 0", h.BackStackSize())
	}

	_, ok = h.Back()
	if ok {
		t.Error("Back should fail when no history exists")
	}
}

func TestNavigationHistory_Forward(t *testing.T) {
	h := NewNavigationHistory[testState](10)

	h.PushToForward(testState{value: "page1"})
	h.PushToForward(testState{value: "page2"})

	state, ok := h.Forward()
	if !ok {
		t.Fatal("Forward should succeed when history exists")
	}

	if state.value != "page2" {
		t.Errorf("Forward returned state.value = %q, want %q", state.value, "page2")
	}

	if h.ForwardStackSize() != 1 {
		t.Errorf("After forward, ForwardStackSize = %d, want 1", h.ForwardStackSize())
	}

	state, ok = h.Forward()
	if !ok {
		t.Fatal("Second forward should succeed")
	}

	if state.value != "page1" {
		t.Errorf("Second forward returned state.value = %q, want %q", state.value, "page1")
	}

	if h.ForwardStackSize() != 0 {
		t.Errorf("After second forward, ForwardStackSize = %d, want 0", h.ForwardStackSize())
	}

	_, ok = h.Forward()
	if ok {
		t.Error("Forward should fail when no history exists")
	}
}

func TestNavigationHistory_MaxSize(t *testing.T) {
	h := NewNavigationHistory[testState](3)

	for i := 1; i <= 5; i++ {
		h.Push(testState{value: "page" + string(rune('0'+i))})
	}

	if h.BackStackSize() != 3 {
		t.Errorf("BackStackSize = %d, want 3 (max size)", h.BackStackSize())
	}

	state1, _ := h.Back()
	if state1.value != "page5" {
		t.Errorf("Latest entry should be page5, got %q", state1.value)
	}
}

func TestNavigationHistory_MaxSizeAlsoAppliesToForwardAndPushToBack(t *testing.T) {
	h := NewNavigationHistory[testState](3)

	// Forward stack should also be trimmed.
	h.PushToForward(testState{value: "f1"})
	h.PushToForward(testState{value: "f2"})
	h.PushToForward(testState{value: "f3"})
	h.PushToForward(testState{value: "f4"})
	h.PushToForward(testState{value: "f5"})

	if h.ForwardStackSize() != 3 {
		t.Fatalf("ForwardStackSize = %d, want 3 (max size)", h.ForwardStackSize())
	}

	// Back stack should be trimmed even when using PushToBack.
	h.PushToBack(testState{value: "b1"})
	h.PushToBack(testState{value: "b2"})
	h.PushToBack(testState{value: "b3"})
	h.PushToBack(testState{value: "b4"})
	h.PushToBack(testState{value: "b5"})

	if h.BackStackSize() != 3 {
		t.Fatalf("BackStackSize = %d, want 3 (max size)", h.BackStackSize())
	}
}
