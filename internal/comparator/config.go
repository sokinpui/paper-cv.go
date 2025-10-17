package comparator

// Config holds all the configuration parameters for the application,
// parsed from command-line flags.
type Config struct {
	InputPath       string
	OutputDirectory string
	UnitSize        int
	Threshold       float64
	CPUCores        int
	ImageType       string
	Device          string
}
