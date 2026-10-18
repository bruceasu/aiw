module aiw-packer

go 1.25.1

require (
	github.com/dsnet/compress v0.0.1
	golang.org/x/text v0.24.0
)

replace github.com/dsnet/compress => ./third_party/dsnet-compress
replace golang.org/x/text => ./third_party/x-text
