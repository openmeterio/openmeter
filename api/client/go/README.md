# OpenMeter Go SDK

## Install

```sh
go get github.com/openmeterio/openmeter
```

## Usage

```go
func main() {
    // Initialize OpenMeter client
    om, err := openmeter.NewClient("http://localhost:8888")
    if err != nil {
        panic(err.Error())
    }

    // Use OpenMeter client
    // ...
}
```
