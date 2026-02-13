package tsql

import (
	"testing"

	"github.com/codegraph-labs/codegraph/internal/parser"
)

func TestParseCreateTable(t *testing.T) {
	input := `
CREATE TABLE dbo.Users (
    UserID INT IDENTITY(1,1) PRIMARY KEY,
    Username NVARCHAR(50) NOT NULL,
    Email NVARCHAR(255) NOT NULL,
    CreatedAt DATETIME2 DEFAULT GETDATE()
);
GO
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "test.sql", Content: []byte(input)})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Symbols) == 0 {
		t.Fatal("expected at least 1 symbol")
	}

	table := result.Symbols[0]
	if table.Kind != "table" {
		t.Errorf("expected table, got %s", table.Kind)
	}
	if table.QualifiedName != "dbo.Users" {
		t.Errorf("expected dbo.Users, got %s", table.QualifiedName)
	}
	if len(table.Children) < 3 {
		t.Errorf("expected at least 3 columns, got %d", len(table.Children))
	}
}

func TestParseCreateProcedure(t *testing.T) {
	input := `
CREATE PROCEDURE dbo.GetUserOrders
    @UserID INT,
    @Status VARCHAR(20) = NULL
AS
BEGIN
    SET NOCOUNT ON;

    SELECT o.OrderID, o.Total
    FROM dbo.Orders o
    WHERE o.UserID = @UserID;

    INSERT INTO dbo.AuditLog (Action, UserID)
    VALUES ('GetOrders', @UserID);

    EXEC dbo.UpdateLastAccess @UserID;
END
GO
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "test.sql", Content: []byte(input)})
	if err != nil {
		t.Fatal(err)
	}

	// Should have the procedure symbol
	var proc *parser.Symbol
	for i, s := range result.Symbols {
		if s.Kind == "procedure" {
			proc = &result.Symbols[i]
			break
		}
	}
	if proc == nil {
		t.Fatal("expected procedure symbol")
	}
	if proc.QualifiedName != "dbo.GetUserOrders" {
		t.Errorf("expected dbo.GetUserOrders, got %s", proc.QualifiedName)
	}

	// Should have references
	refTypes := map[string]bool{}
	for _, ref := range result.References {
		refTypes[ref.ReferenceType+":"+ref.ToQualified] = true
	}

	if !refTypes["reads_from:dbo.Orders"] {
		t.Error("expected reads_from reference to dbo.Orders")
	}
	if !refTypes["writes_to:dbo.AuditLog"] {
		t.Error("expected writes_to reference to dbo.AuditLog")
	}
	if !refTypes["calls:dbo.UpdateLastAccess"] {
		t.Error("expected calls reference to dbo.UpdateLastAccess")
	}
}

func TestParseCreateView(t *testing.T) {
	input := `
CREATE VIEW dbo.ActiveUsers AS
SELECT u.UserID, u.Username, u.Email
FROM dbo.Users u
WHERE u.IsActive = 1;
GO
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "test.sql", Content: []byte(input)})
	if err != nil {
		t.Fatal(err)
	}

	var view *parser.Symbol
	for i, s := range result.Symbols {
		if s.Kind == "view" {
			view = &result.Symbols[i]
			break
		}
	}
	if view == nil {
		t.Fatal("expected view symbol")
	}
	if view.QualifiedName != "dbo.ActiveUsers" {
		t.Errorf("expected dbo.ActiveUsers, got %s", view.QualifiedName)
	}

	// View should reference Users table
	found := false
	for _, ref := range result.References {
		if ref.ToQualified == "dbo.Users" && ref.ReferenceType == "reads_from" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected reads_from reference to dbo.Users")
	}
}

func TestParseCreateTrigger(t *testing.T) {
	input := `
CREATE TRIGGER dbo.trg_OrderInsert
ON dbo.Orders
AFTER INSERT
AS
BEGIN
    INSERT INTO dbo.OrderHistory (OrderID, Action)
    SELECT i.OrderID, 'INSERT'
    FROM inserted i;
END
GO
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "test.sql", Content: []byte(input)})
	if err != nil {
		t.Fatal(err)
	}

	var trigger *parser.Symbol
	for i, s := range result.Symbols {
		if s.Kind == "trigger" {
			trigger = &result.Symbols[i]
			break
		}
	}
	if trigger == nil {
		t.Fatal("expected trigger symbol")
	}
	if trigger.QualifiedName != "dbo.trg_OrderInsert" {
		t.Errorf("expected dbo.trg_OrderInsert, got %s", trigger.QualifiedName)
	}

	// Should reference Orders (ON table) and OrderHistory (INSERT INTO)
	refTypes := map[string]bool{}
	for _, ref := range result.References {
		refTypes[ref.ReferenceType+":"+ref.ToQualified] = true
	}
	if !refTypes["uses_table:dbo.Orders"] {
		t.Error("expected uses_table reference to dbo.Orders")
	}
	if !refTypes["writes_to:dbo.OrderHistory"] {
		t.Error("expected writes_to reference to dbo.OrderHistory")
	}
}

func TestDialectDetection(t *testing.T) {
	tsql := `
DECLARE @UserID INT = 1;
SELECT TOP 10 * FROM dbo.Users WITH (NOLOCK);
GO
`
	if d := parser.DetectDialect([]byte(tsql)); d != "tsql" {
		t.Errorf("expected tsql, got %s", d)
	}
}
