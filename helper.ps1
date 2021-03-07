$TIMESTAMP = (date -u +%Y%m%d.%H%M%S)
$COVER_PROFILE_FILENAME = "logfeller-testcov"

$target = $args[0]

switch ($target) {
    generate {
        go generate ./...
    }
    test {
        go generate ./...
        $packages = If ($args[1] -eq $null) { "./..." } else { $args[1] }
        go test -cover $packages
    }
    showtestcov {
        go generate ./...
        $packages = If ($args[1] -eq $null) { "./..." } else { $args[1] }
        $fullcoverpath = "$env:TEMP\$COVER_PROFILE_FILENAME.$TIMESTAMP.out"
        Write-Output "Running tests over $packages and saving output to $fullcoverpath"
        go test $packages -covermode count -coverprofile $fullcoverpath
        if(!$?) {
           exit $?
        }
        go tool cover -html="$fullcoverpath"
    }
    lint {
        go generate ./...
        golangci-lint --version
        golangci-lint run
    }
    Default {
        Write-Output "usage: powershell .\helper.ps1 (generate|test|showtestcov|lint)"
    }
}