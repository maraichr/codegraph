package java

import (
	"testing"

	"github.com/maraichr/lattice/internal/parser"
)

func TestBasicClass(t *testing.T) {
	src := `
package com.example;

import java.util.List;

public class User {
    private String name;
    public String getName() { return name; }
    public User() {}
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "User.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	assertHasSymbol(t, result.Symbols, "com.example.User", "class")
	assertHasSymbol(t, result.Symbols, "com.example.User.getName", "method")
	assertHasRef(t, result.References, "java.util.List", "imports")
}

func TestEntityAnnotation(t *testing.T) {
	src := `
package com.example;

@Entity
@Table(name = "users")
public class User {
    @Id
    private Long id;
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "User.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	tableRefs := filterRefs(result.References, "uses_table")
	assertRefTarget(t, tableRefs, "users")
}

func TestQueryAnnotation(t *testing.T) {
	src := `
package com.example;

public interface UserRepository {
    @Query("SELECT u FROM Users u WHERE u.active = true")
    List<User> findActiveUsers();
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "UserRepository.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	tableRefs := filterRefs(result.References, "uses_table")
	assertRefTarget(t, tableRefs, "Users")
}

func TestJDBCPrepareStatement(t *testing.T) {
	src := `
package com.example;

public class UserDao {
    public User getById(int id) {
        PreparedStatement ps = conn.prepareStatement("SELECT * FROM users WHERE id = ?");
        return mapResult(ps.executeQuery());
    }
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "UserDao.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	tableRefs := filterRefs(result.References, "uses_table")
	assertRefTarget(t, tableRefs, "users")
}

func TestJDBCPrepareCall(t *testing.T) {
	src := `
package com.example;

public class UserDao {
    public void callProc() {
        CallableStatement cs = conn.prepareCall("EXEC dbo.GetUser ?");
    }
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "UserDao.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	callRefs := filterRefs(result.References, "calls")
	assertRefTarget(t, callRefs, "dbo.GetUser")
}

func TestSpringDataRepository(t *testing.T) {
	src := `
package com.example;

public interface UserRepository extends JpaRepository<User, Long> {
    List<User> findByEmailAndStatus(String email, String status);
    long countByStatus(String status);
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "UserRepository.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	tableRefs := filterRefs(result.References, "uses_table")
	// Should have uses_table for User (from JpaRepository<User, Long>)
	assertRefTarget(t, tableRefs, "User")

	// Derived query methods should also reference User
	count := 0
	for _, r := range tableRefs {
		if r.ToName == "User" {
			count++
		}
	}
	// 1 from JpaRepository<User, Long> + 2 from derived query methods
	if count < 3 {
		t.Errorf("expected at least 3 uses_table refs for User, got %d", count)
	}
}

func TestNamedQuery(t *testing.T) {
	src := `
package com.example;

@NamedQuery(name = "User.findAll", query = "SELECT u FROM Users u")
public class User {
    private String name;
}
`
	p := New()
	result, err := p.Parse(parser.FileInput{Path: "User.java", Content: []byte(src)})
	if err != nil {
		t.Fatal(err)
	}

	tableRefs := filterRefs(result.References, "uses_table")
	assertRefTarget(t, tableRefs, "Users")
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

func assertHasRef(t *testing.T, refs []parser.RawReference, toName, refType string) {
	t.Helper()
	for _, r := range refs {
		if (r.ToName == toName || r.ToQualified == toName) && r.ReferenceType == refType {
			return
		}
	}
	t.Errorf("missing ref %s (%s)", toName, refType)
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
