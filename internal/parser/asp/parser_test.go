package asp

import (
	"testing"

	"github.com/maraichr/codegraph/internal/parser"
)

func TestDirectivesCodeBehind(t *testing.T) {
	src := `<%@ Page Language="C#" CodeBehind="Users.aspx.cs" Inherits="MyApp.UsersPage" %>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "Users.aspx", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	imports := filterRefs(result.References, "imports")
	assertRefTarget(t, imports, "Users.aspx.cs")

	inherits := filterRefs(result.References, "inherits")
	assertRefTarget(t, inherits, "MyApp.UsersPage")
}

func TestDirectivesImportNamespace(t *testing.T) {
	src := `<%@ Import Namespace="System.Data" %>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "Page.aspx", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	imports := filterRefs(result.References, "imports")
	assertRefTarget(t, imports, "System.Data")
}

func TestDirectivesRegister(t *testing.T) {
	src := `<%@ Register Assembly="Telerik.Web.UI" Namespace="Telerik.Web.UI" TagPrefix="telerik" %>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "Page.aspx", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	imports := filterRefs(result.References, "imports")
	assertRefTarget(t, imports, "Telerik.Web.UI")
}

func TestDirectivesControlCodeBehind(t *testing.T) {
	src := `<%@ Control Language="C#" CodeBehind="UserControl.ascx.cs" Inherits="MyApp.UserControl" %>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "UserControl.ascx", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	imports := filterRefs(result.References, "imports")
	assertRefTarget(t, imports, "UserControl.ascx.cs")

	inherits := filterRefs(result.References, "inherits")
	assertRefTarget(t, inherits, "MyApp.UserControl")
}

func TestDirectivesMixedWithVBScript(t *testing.T) {
	src := `<%@ Page Language="C#" CodeBehind="Users.aspx.cs" Inherits="MyApp.UsersPage" %>
<%@ Import Namespace="System.Data" %>
<html>
<% Dim x = 1 %>
</html>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "Users.aspx", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	imports := filterRefs(result.References, "imports")
	assertRefTarget(t, imports, "Users.aspx.cs")
	assertRefTarget(t, imports, "System.Data")

	inherits := filterRefs(result.References, "inherits")
	assertRefTarget(t, inherits, "MyApp.UsersPage")
}

func TestVBScriptFunction(t *testing.T) {
	src := `<%
Function GetUserName(userId)
  GetUserName = "test"
End Function
%>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "utils.asp", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	assertHasSymbol(t, result.Symbols, "GetUserName", "function")
}

func TestVBScriptSub(t *testing.T) {
	src := `<%
Sub ProcessData()
  ' do something
End Sub
%>`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "process.asp", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	assertHasSymbol(t, result.Symbols, "ProcessData", "procedure")
}

func TestIncludeDirective(t *testing.T) {
	src := `<!-- #include file="header.asp" -->`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "page.asp", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	imports := filterRefs(result.References, "imports")
	assertRefTarget(t, imports, "header.asp")
}

func TestLanguages(t *testing.T) {
	p := New()
	langs := p.Languages()
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d: %v", len(langs), langs)
	}
}

// --- helpers ---

func assertHasSymbol(t *testing.T, symbols []parser.Symbol, qname, kind string) {
	t.Helper()
	for _, s := range symbols {
		if s.QualifiedName == qname && s.Kind == kind {
			return
		}
	}
	names := make([]string, len(symbols))
	for i, s := range symbols {
		names[i] = s.QualifiedName + " (" + s.Kind + ")"
	}
	t.Errorf("missing symbol %s (%s); have: %v", qname, kind, names)
}

func filterRefs(refs []parser.RawReference, refType string) []parser.RawReference {
	var out []parser.RawReference
	for _, r := range refs {
		if r.ReferenceType == refType {
			out = append(out, r)
		}
	}
	return out
}

func assertRefTarget(t *testing.T, refs []parser.RawReference, target string) {
	t.Helper()
	for _, r := range refs {
		if r.ToName == target || r.ToQualified == target {
			return
		}
	}
	names := make([]string, len(refs))
	for i, r := range refs {
		names[i] = r.ToName
	}
	t.Errorf("missing ref target %s; have: %v", target, names)
}
