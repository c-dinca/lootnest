package app

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// TestNoLockInsideLock reads this package's source and fails when code that
// holds a.mu calls an App method that takes a.mu itself, directly or through
// other methods. Go's mutex isn't re-entrant, so such a call freezes
// Lootnest: from 2.19.0 to 2.24.0 runSwap asked launcherWait, which asks
// launchOpts, under the lock, and every start of a game with profiles hung
// the window and left the session "launching".
//
// The check is lexical: a region runs from a.mu.Lock() to the next
// a.mu.Unlock() in the same statement list, or to the end of the list after
// a deferred Unlock. Statements that unlock inside themselves end what's
// checked in them, and goroutines and function literals called later are
// left out, so it misses some cases but reports none falsely.
func TestNoLockInsideLock(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	pkg := pkgs["app"]
	if pkg == nil {
		t.Fatal("package app not found")
	}

	type method struct {
		body  *ast.BlockStmt
		locks bool
		calls map[string]bool
	}
	methods := map[string]*method{}
	var bodies []*ast.FuncDecl
	for _, f := range pkg.Files {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			bodies = append(bodies, fd)
			if !onApp(fd) {
				continue
			}
			m := &method{body: fd.Body, calls: map[string]bool{}}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.FuncLit, *ast.GoStmt:
					return false // runs later or elsewhere
				case *ast.CallExpr:
					if muCall(n, "Lock") {
						m.locks = true
					}
					if name := appCall(n); name != "" {
						m.calls[name] = true
					}
				}
				return true
			})
			methods[fd.Name.Name] = m
		}
	}
	takes := map[string]bool{}
	for name, m := range methods {
		takes[name] = m.locks
	}
	for changed := true; changed; {
		changed = false
		for name, m := range methods {
			if takes[name] {
				continue
			}
			for c := range m.calls {
				if takes[c] {
					takes[name], changed = true, true
					break
				}
			}
		}
	}

	check := func(fd *ast.FuncDecl, n ast.Node, at token.Pos) {
		ast.Inspect(n, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.FuncLit, *ast.GoStmt:
				return false
			case *ast.CallExpr:
				if muCall(n, "Lock") {
					t.Errorf("%s: %s takes a.mu again while holding it (taken at %s)", fset.Position(n.Pos()), fd.Name.Name, fset.Position(at))
				} else if name := appCall(n); name != "" && takes[name] {
					t.Errorf("%s: %s calls a.%s, which takes a.mu, while holding it (taken at %s)", fset.Position(n.Pos()), fd.Name.Name, name, fset.Position(at))
				}
			}
			return true
		})
	}
	unlocks := func(n ast.Node) bool {
		found := false
		ast.Inspect(n, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			if c, ok := n.(*ast.CallExpr); ok && muCall(c, "Unlock") {
				found = true
			}
			return !found
		})
		return found
	}
	var walk func(fd *ast.FuncDecl, list []ast.Stmt)
	walk = func(fd *ast.FuncDecl, list []ast.Stmt) {
		for i := 0; i < len(list); i++ {
			if es, ok := list[i].(*ast.ExprStmt); ok && muCall(es.X, "Lock") {
				at, j, deferred := es.Pos(), i+1, false
				if j < len(list) {
					if ds, ok := list[j].(*ast.DeferStmt); ok && muCall(ds.Call, "Unlock") {
						deferred = true
						j++
					}
				}
				for ; j < len(list); j++ {
					if !deferred && unlocks(list[j]) {
						if es, ok := list[j].(*ast.ExprStmt); ok && muCall(es.X, "Unlock") {
							break // the region ends here
						}
						continue // unlocks somewhere inside: not followed
					}
					check(fd, list[j], at)
				}
				i = j
				continue
			}
			ast.Inspect(list[i], func(n ast.Node) bool {
				switch b := n.(type) {
				case *ast.BlockStmt:
					walk(fd, b.List)
					return false
				case *ast.CaseClause:
					walk(fd, b.Body)
					return false
				case *ast.CommClause:
					walk(fd, b.Body)
					return false
				}
				return true
			})
		}
	}
	for _, fd := range bodies {
		walk(fd, fd.Body.List)
	}
}

// onApp is a method on *App.
func onApp(fd *ast.FuncDecl) bool {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return false
	}
	st, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	id, ok := st.X.(*ast.Ident)
	return ok && id.Name == "App"
}

// muCall is a.mu.<name>().
func muCall(e ast.Expr, name string) bool {
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	mu, ok := sel.X.(*ast.SelectorExpr)
	if !ok || mu.Sel.Name != "mu" {
		return false
	}
	id, ok := mu.X.(*ast.Ident)
	return ok && id.Name == "a"
}

// appCall is the method's name in a call a.<name>(...), else "".
func appCall(c *ast.CallExpr) string {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	if id, ok := sel.X.(*ast.Ident); ok && id.Name == "a" {
		return sel.Sel.Name
	}
	return ""
}
