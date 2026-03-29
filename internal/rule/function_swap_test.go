package rule

import "testing"

func TestParsePackageFunctionNameAllowsDotInLastPackageElement(t *testing.T) {
	pkgPath, functionName, ok := parsePackageFunctionName("gopkg.in/yaml.v3.Marshal")
	if !ok {
		t.Fatal("parsePackageFunctionName() should accept package element with dot")
	}
	if pkgPath != "gopkg.in/yaml.v3" {
		t.Fatalf("pkgPath = %q, want %q", pkgPath, "gopkg.in/yaml.v3")
	}
	if functionName != "Marshal" {
		t.Fatalf("functionName = %q, want %q", functionName, "Marshal")
	}
}
