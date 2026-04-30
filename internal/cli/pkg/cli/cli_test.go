package cli

import "testing"

func TestCli_ExactArgs_Good(t *testing.T) {
	validator := ExactArgs(2)
	err := validator(&Command{}, []string{"one", "two"})

	if err != nil {
		t.Fatalf("expected exact args to pass, got %v", err)
	}
}

func TestCli_ExactArgs_Bad(t *testing.T) {
	validator := ExactArgs(2)
	err := validator(&Command{}, []string{"one"})

	if err == nil {
		t.Fatalf("expected exact args to reject wrong count")
	}
}

func TestCli_ExactArgs_Ugly(t *testing.T) {
	validator := ExactArgs(0)
	err := validator(&Command{}, nil)

	if err != nil {
		t.Fatalf("expected zero exact args to pass, got %v", err)
	}
}

func TestCli_MaximumNArgs_Good(t *testing.T) {
	validator := MaximumNArgs(2)
	err := validator(&Command{}, []string{"one"})

	if err != nil {
		t.Fatalf("expected args below maximum to pass, got %v", err)
	}
}

func TestCli_MaximumNArgs_Bad(t *testing.T) {
	validator := MaximumNArgs(1)
	err := validator(&Command{}, []string{"one", "two"})

	if err == nil {
		t.Fatalf("expected args above maximum to fail")
	}
}

func TestCli_MaximumNArgs_Ugly(t *testing.T) {
	validator := MaximumNArgs(0)
	err := validator(&Command{}, nil)

	if err != nil {
		t.Fatalf("expected nil args to satisfy zero maximum, got %v", err)
	}
}

func TestCli_NewGroup_Good(t *testing.T) {
	cmd := NewGroup("rag", "short", "long")

	if cmd.Use != "rag" {
		t.Fatalf("want use rag, got %s", cmd.Use)
	}
	if cmd.Long != "long" {
		t.Fatalf("want long text, got %s", cmd.Long)
	}
}

func TestCli_NewGroup_Bad(t *testing.T) {
	cmd := NewGroup("", "", "")

	if cmd.Use != "" {
		t.Fatalf("want empty use, got %s", cmd.Use)
	}
	if cmd.Long != "" {
		t.Fatalf("want empty long, got %s", cmd.Long)
	}
}

func TestCli_NewGroup_Ugly(t *testing.T) {
	cmd := NewGroup("rag", "", "")

	if cmd.Short != "" {
		t.Fatalf("want empty short, got %s", cmd.Short)
	}
	if len(cmd.Commands()) != 0 {
		t.Fatalf("want no child commands, got %d", len(cmd.Commands()))
	}
}

func TestCli_Style_Render_Good(t *testing.T) {
	style := Style{}
	got := style.Render("Collections")

	if got != "Collections" {
		t.Fatalf("want rendered text unchanged, got %s", got)
	}
}

func TestCli_Style_Render_Bad(t *testing.T) {
	style := Style{}
	got := style.Render("")

	if got != "" {
		t.Fatalf("want empty text unchanged, got %s", got)
	}
}

func TestCli_Style_Render_Ugly(t *testing.T) {
	style := Style{}
	got := style.Render("emoji 😀")

	if got != "emoji 😀" {
		t.Fatalf("want unicode text unchanged, got %s", got)
	}
}
