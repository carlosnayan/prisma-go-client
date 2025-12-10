package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateLoggerInline_ContainsRequiredMethods(t *testing.T) {
	// Create a temporary directory for generated code
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	builderDir := filepath.Join(tmpDir, "builder")
	err = os.MkdirAll(builderDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create builder dir: %v", err)
	}

	// Generate the builder fluent file
	err = generateBuilderFluent(builderDir, "postgresql", "test/utils")
	if err != nil {
		t.Fatalf("Failed to generate builder fluent: %v", err)
	}

	// Read the generated file
	fluentPath := filepath.Join(builderDir, "fluent.go")
	content, err := os.ReadFile(fluentPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify Logger struct has required fields
	if !strings.Contains(contentStr, "type Logger struct") {
		t.Error("Generated code should contain Logger struct")
	}
	if !strings.Contains(contentStr, "levels   map[string]bool") {
		t.Error("Logger struct should contain levels map[string]bool field")
	}
	if !strings.Contains(contentStr, "queryLog bool") {
		t.Error("Logger struct should contain queryLog bool field")
	}

	// Verify defaultLogger initialization
	if !strings.Contains(contentStr, "var defaultLogger = &Logger{") {
		t.Error("Generated code should contain defaultLogger variable")
	}
	if !strings.Contains(contentStr, "levels:   make(map[string]bool),") {
		t.Error("defaultLogger should initialize levels map")
	}

	// Verify Query method exists and logs
	if !strings.Contains(contentStr, "func (l *Logger) Query(") {
		t.Error("Logger should have Query method")
	}
	if !strings.Contains(contentStr, `fmt.Printf("[%s] [QUERY]`) {
		t.Error("Query method should log with [QUERY] prefix")
	}

	// Verify Info method exists and logs
	if !strings.Contains(contentStr, "func (l *Logger) Info(") {
		t.Error("Logger should have Info method")
	}
	if !strings.Contains(contentStr, `fmt.Printf("[%s] [INFO]`) {
		t.Error("Info method should log with [INFO] prefix")
	}
	if !strings.Contains(contentStr, `l.levels["info"]`) {
		t.Error("Info method should check levels[\"info\"]")
	}

	// Verify Warn method exists and logs
	if !strings.Contains(contentStr, "func (l *Logger) Warn(") {
		t.Error("Logger should have Warn method")
	}
	if !strings.Contains(contentStr, `fmt.Printf("[%s] [WARN]`) {
		t.Error("Warn method should log with [WARN] prefix")
	}
	if !strings.Contains(contentStr, `l.levels["warn"]`) {
		t.Error("Warn method should check levels[\"warn\"]")
	}

	// Verify Error method exists and logs
	if !strings.Contains(contentStr, "func (l *Logger) Error(") {
		t.Error("Logger should have Error method")
	}
	if !strings.Contains(contentStr, `fmt.Printf("[%s] [ERROR]`) {
		t.Error("Error method should log with [ERROR] prefix")
	}
	if !strings.Contains(contentStr, `l.levels["error"]`) {
		t.Error("Error method should check levels[\"error\"]")
	}

	// Verify SetLogLevels function exists and configures all levels
	if !strings.Contains(contentStr, "func SetLogLevels(levels []string)") {
		t.Error("Generated code should contain SetLogLevels function")
	}
	if !strings.Contains(contentStr, "defaultLogger.levels = make(map[string]bool)") {
		t.Error("SetLogLevels should initialize levels map")
	}
	if !strings.Contains(contentStr, `strings.ToLower(strings.TrimSpace(level))`) {
		t.Error("SetLogLevels should normalize level names")
	}
	if !strings.Contains(contentStr, `defaultLogger.levels[levelLower] = true`) {
		t.Error("SetLogLevels should set levels in the map")
	}
	if !strings.Contains(contentStr, `if levelLower == "query"`) {
		t.Error("SetLogLevels should handle query level specially")
	}

	// Verify GetDefaultLogger function exists
	if !strings.Contains(contentStr, "func GetDefaultLogger() *Logger") {
		t.Error("Generated code should contain GetDefaultLogger function")
	}
}

func TestGenerateLoggerInline_LogMethodsNotEmpty(t *testing.T) {
	// Create a temporary directory for generated code
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	builderDir := filepath.Join(tmpDir, "builder")
	err = os.MkdirAll(builderDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create builder dir: %v", err)
	}

	// Generate the builder fluent file
	err = generateBuilderFluent(builderDir, "postgresql", "test/utils")
	if err != nil {
		t.Fatalf("Failed to generate builder fluent: %v", err)
	}

	// Read the generated file
	fluentPath := filepath.Join(builderDir, "fluent.go")
	content, err := os.ReadFile(fluentPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify that log methods are not empty (not just comments)
	// They should contain actual fmt.Printf calls, not just "// Simplified logging"

	// Check that Info method has actual implementation
	infoStart := strings.Index(contentStr, "func (l *Logger) Info(")
	if infoStart == -1 {
		t.Fatal("Info method not found")
	}
	infoEnd := strings.Index(contentStr[infoStart:], "}\n\n")
	if infoEnd == -1 {
		t.Fatal("Info method end not found")
	}
	infoMethod := contentStr[infoStart : infoStart+infoEnd]
	if strings.Contains(infoMethod, "// Simplified logging - can be extended if needed") {
		t.Error("Info method should not be empty - it should have actual logging implementation")
	}
	if !strings.Contains(infoMethod, "fmt.Printf") {
		t.Error("Info method should contain fmt.Printf call")
	}

	// Check that Warn method has actual implementation
	warnStart := strings.Index(contentStr, "func (l *Logger) Warn(")
	if warnStart == -1 {
		t.Fatal("Warn method not found")
	}
	warnEnd := strings.Index(contentStr[warnStart:], "}\n\n")
	if warnEnd == -1 {
		t.Fatal("Warn method end not found")
	}
	warnMethod := contentStr[warnStart : warnStart+warnEnd]
	if strings.Contains(warnMethod, "// Simplified logging - can be extended if needed") {
		t.Error("Warn method should not be empty - it should have actual logging implementation")
	}
	if !strings.Contains(warnMethod, "fmt.Printf") {
		t.Error("Warn method should contain fmt.Printf call")
	}

	// Check that Error method has actual implementation
	errorStart := strings.Index(contentStr, "func (l *Logger) Error(")
	if errorStart == -1 {
		t.Fatal("Error method not found")
	}
	errorEnd := strings.Index(contentStr[errorStart:], "}\n\n")
	if errorEnd == -1 {
		t.Fatal("Error method end not found")
	}
	errorMethod := contentStr[errorStart : errorStart+errorEnd]
	if strings.Contains(errorMethod, "// Simplified logging - can be extended if needed") {
		t.Error("Error method should not be empty - it should have actual logging implementation")
	}
	if !strings.Contains(errorMethod, "fmt.Printf") {
		t.Error("Error method should contain fmt.Printf call")
	}

	// Check that Query method has actual implementation
	queryStart := strings.Index(contentStr, "func (l *Logger) Query(")
	if queryStart == -1 {
		t.Fatal("Query method not found")
	}
	queryEnd := strings.Index(contentStr[queryStart:], "}\n\n")
	if queryEnd == -1 {
		t.Fatal("Query method end not found")
	}
	queryMethod := contentStr[queryStart : queryStart+queryEnd]
	if strings.Contains(queryMethod, "// Simplified logging - can be extended if needed") {
		t.Error("Query method should not be empty - it should have actual logging implementation")
	}
	if !strings.Contains(queryMethod, "fmt.Printf") {
		t.Error("Query method should contain fmt.Printf call")
	}
}

func TestGenerateLoggerInline_LogQueryWithTimingCallsInfo(t *testing.T) {
	// Create a temporary directory for generated code
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	builderDir := filepath.Join(tmpDir, "builder")
	err = os.MkdirAll(builderDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create builder dir: %v", err)
	}

	// Generate the builder fluent file
	err = generateBuilderFluent(builderDir, "postgresql", "test/utils")
	if err != nil {
		t.Fatalf("Failed to generate builder fluent: %v", err)
	}

	// Read the generated file
	fluentPath := filepath.Join(builderDir, "fluent.go")
	content, err := os.ReadFile(fluentPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify that logQueryWithTiming calls logger.Info()
	if !strings.Contains(contentStr, "logger.Info(") {
		t.Error("logQueryWithTiming should call logger.Info()")
	}

	// Verify that logQueryWithTiming calls logger.Warn() for slow queries
	if !strings.Contains(contentStr, "logger.Warn(") {
		t.Error("logQueryWithTiming should call logger.Warn() for slow queries")
	}

	// Verify that detectQueryType function exists
	if !strings.Contains(contentStr, "func detectQueryType(query string) string") {
		t.Error("logQueryWithTiming should use detectQueryType function")
	}

	// Verify that it logs query type, duration, overhead and total time
	if !strings.Contains(contentStr, `logger.Info("%s query: %v, overhead: %v (total: %v)"`) {
		t.Error("logQueryWithTiming should log query type, duration, overhead and total time")
	}

	// Verify that it checks for slow queries (> 1000ms)
	if !strings.Contains(contentStr, "if queryDuration > 1000*time.Millisecond") {
		t.Error("logQueryWithTiming should check for slow queries (> 1000ms)")
	}
}

func TestGenerateLoggerInline_GetLoggerUsesCurrentDefault(t *testing.T) {
	// Create a temporary directory for generated code
	tmpDir, err := os.MkdirTemp("", "prisma-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	builderDir := filepath.Join(tmpDir, "builder")
	err = os.MkdirAll(builderDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create builder dir: %v", err)
	}

	// Generate the builder fluent file
	err = generateBuilderFluent(builderDir, "postgresql", "test/utils")
	if err != nil {
		t.Fatalf("Failed to generate builder fluent: %v", err)
	}

	// Read the generated file
	fluentPath := filepath.Join(builderDir, "fluent.go")
	content, err := os.ReadFile(fluentPath)
	if err != nil {
		t.Fatalf("Failed to read generated file: %v", err)
	}

	contentStr := string(content)

	// Verify getLogger() always gets the current default logger
	if !strings.Contains(contentStr, "func (q *Query) getLogger() *Logger") {
		t.Error("Generated code should contain getLogger() method")
	}

	// Find getLogger method
	getLoggerStart := strings.Index(contentStr, "func (q *Query) getLogger() *Logger")
	if getLoggerStart == -1 {
		t.Fatal("getLogger method not found")
	}
	getLoggerEnd := strings.Index(contentStr[getLoggerStart:], "}\n\n")
	if getLoggerEnd == -1 {
		t.Fatal("getLogger method end not found")
	}
	getLoggerMethod := contentStr[getLoggerStart : getLoggerStart+getLoggerEnd]

	// Verify it calls GetDefaultLogger()
	if !strings.Contains(getLoggerMethod, "GetDefaultLogger()") {
		t.Error("getLogger() should call GetDefaultLogger() to get current logger")
	}

	// Verify it updates q.logger if different
	if !strings.Contains(getLoggerMethod, "currentLogger := GetDefaultLogger()") {
		t.Error("getLogger() should get current default logger")
	}
	if !strings.Contains(getLoggerMethod, "if currentLogger != q.logger") {
		t.Error("getLogger() should check if current logger is different from q.logger")
	}
	if !strings.Contains(getLoggerMethod, "q.logger = currentLogger") {
		t.Error("getLogger() should update q.logger if different")
	}

	// Verify it returns q.logger (which is now the current logger)
	if !strings.Contains(getLoggerMethod, "return q.logger") {
		t.Error("getLogger() should return q.logger")
	}
}
