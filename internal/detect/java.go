package detect

import (
	"github.com/nqode/qode/internal/config"
)

// JavaDetector identifies a Java/Kotlin project.
type JavaDetector struct{}

func (d *JavaDetector) Name() string { return "java" }

func (d *JavaDetector) Detect(root string) (bool, float64) {
	// Maven.
	if fileExists(root, "pom.xml") {
		return true, 1.0
	}
	// Gradle (Groovy or Kotlin DSL).
	if fileExists(root, "build.gradle") || fileExists(root, "build.gradle.kts") {
		return true, 0.95
	}
	// Gradle wrapper.
	if fileExists(root, "gradlew") || fileExists(root, "gradlew.bat") {
		return true, 0.85
	}
	return false, 0
}

func (d *JavaDetector) DefaultConfig() config.TestConfig {
	return config.StackDefaults["java"]
}
