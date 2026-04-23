package version

var (
	BuildDate  = "unknown"
	GitCommit  = "unknown"
	GitVersion = "dev"
)

func GetBuildInfo() (buildDate, gitCommit, gitVersion string) {
	return BuildDate, GitCommit, GitVersion
}