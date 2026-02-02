package cmd

import "testing"

// Export internal functions and variables for testing in external packages

func ResetGlobalFlags() {
	resetGlobalFlags()
}

func SetupTestDir(t testing.TB) string {
	return setupTestDir(t)
}

func CleanupTestDir(dir string) error {
	return cleanupTestDir(dir)
}

func RunInit(args []string) error {
	return runInit(args)
}

func SetProviderFlag(v string) {
	providerFlag = v
}

func SetDatabaseFlag(v string) {
	databaseFlag = v
}

func SetSchemaPath(v string) {
	schemaPath = v
}

func FileExists(path string) bool {
	return fileExists(path)
}

func AssertDirExists(t testing.TB, path string) {
	assertDirExists(t, path)
}

func ReadFile(t testing.TB, path string) string {
	return readFile(t, path)
}

func Contains(s, substr string) bool {
	return contains(s, substr)
}

// Generate Flags and Run
func SetGeneratorFlags(v []string) {
	generatorFlags = v
}
func SetWatchFlag(v bool) {
	watchFlag = v
}
func SetNoHintsFlag(v bool) {
	noHintsFlag = v
}
func SetRequireModelsFlag(v bool) {
	requireModelsFlag = v
}
func RunGenerate(args []string) error {
	return runGenerate(args)
}

// Migrate Flags and Run
func SetMigrateResolveAppliedFlag(v string) {
	migrateResolveAppliedFlag = v
}
func SetMigrateResolveRolledBackFlag(v string) {
	migrateResolveRolledBackFlag = v
}
func RunMigrateDev(args []string) error {
	return runMigrateDev(args)
}
func RunMigrateDeploy(args []string) error {
	return runMigrateDeploy(args)
}
func RunMigrateReset(args []string) error {
	return runMigrateReset(args)
}
func RunMigrateStatus(args []string) error {
	return runMigrateStatus(args)
}
func RunMigrateResolve(args []string) error {
	return runMigrateResolve(args)
}
func RunMigrateDiff(args []string) error {
	return runMigrateDiff(args)
}

// Migrate Diff Flags
func SetDiffFrom(v string) {
	diffFrom = v
}
func SetDiffTo(v string) {
	diffTo = v
}
func SetDiffOut(v string) {
	diffOut = v
}

// DB Flags and Run
func SetDbPushAcceptDataLossFlag(v bool) {
	dbPushAcceptDataLossFlag = v
}
func SetDbPushSkipGenerateFlag(v bool) {
	dbPushSkipGenerateFlag = v
}
func SetDbExecuteFileFlag(v string) {
	dbExecuteFileFlag = v
}
func SetDbExecuteStdinFlag(v bool) {
	dbExecuteStdinFlag = v
}
func RunDbPush(args []string) error {
	return runDbPush(args)
}
func RunDbPull(args []string) error {
	return runDbPull(args)
}
func RunDbSeed(args []string) error {
	return runDbSeed(args)
}
func RunDbExecute(args []string) error {
	return runDbExecute(args)
}

// Validate and Format Flags and Run
func SetFormatCheckFlag(v bool) {
	formatCheckFlag = v
}
func RunValidate(args []string) error {
	return runValidate(args)
}
func RunFormat(args []string) error {
	return runFormat(args)
}

func ParseGeneratorFlags(args []string) []string {
	return parseGeneratorFlags(args)
}

func CreateTestGoMod(t testing.TB, moduleName string) string {
	return createTestGoMod(t, moduleName)
}

func CreateTestConfig(t testing.TB, content string) string {
	return createTestConfig(t, content)
}

func CreateTestSchema(t testing.TB, content string) string {
	return createTestSchema(t, content)
}

func CreateInvalidSchema(t testing.TB) string {
	return createInvalidSchema(t)
}

func SetEnv(t testing.TB, key, value string) func() {
	return setEnv(t, key, value)
}

func SkipIfNoDatabase(t *testing.T) {
	skipIfNoDatabase(t)
}

func GetDbExecuteFileFlag() string {
	return dbExecuteFileFlag
}

func GetDbExecuteStdinFlag() bool {
	return dbExecuteStdinFlag
}

func CreateIsolatedTestDB(t *testing.T) (string, func()) {
	return createIsolatedTestDB(t)
}

func GetTestDBURL(t *testing.T, dbName string) string {
	return getTestDBURL(t, dbName)
}

func ExecSQL(t *testing.T, dbURL string, sqlStatements ...string) {
	execSQL(t, dbURL, sqlStatements...)
}

func TableExists(t *testing.T, dbURL, tableName string) bool {
	return tableExists(t, dbURL, tableName)
}

func GetTestDatabaseURL(t testing.TB) string {
	return getTestDatabaseURL(t)
}

func CreateTestMigration(t testing.TB, name, sql string) string {
	return createTestMigration(t, name, sql)
}

func GetConfigFile() string {
	return configFile
}

func SetConfigFile(v string) {
	configFile = v
}
