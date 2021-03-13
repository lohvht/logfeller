$TIMESTAMP = (date -u +%Y%m%d.%H%M%S)
$COVER_PROFILE_FILENAME = "$env:TEMP\logfeller-testcov.$TIMESTAMP.out"

$target = $args[0]

switch ($target) {
    generate {
        go generate ./...
    }
    test {
        go generate ./...
        $packages = If ($args[1] -eq $null) { "./..." } else { $args[1] }
        go test -cover -race $packages
    }
    testcov {
        go generate ./...
        $packages = If ($args[1] -eq $null) { "./..." } else { $args[1] }
        $coverfilepath = If ($args[2] -eq $null) { $COVER_PROFILE_FILENAME } else { $args[2] }

        Write-Output "Running tests over $packages and saving output to $coverfilepath"
        go test -race $packages -covermode atomic -coverprofile $coverfilepath
        if(!$?) {
           exit $?
        }
    }
    showtestcov {
        go generate ./...
        $packages = If ($args[1] -eq $null) { "./..." } else { $args[1] }
        $coverfilepath = If ($args[2] -eq $null) { $COVER_PROFILE_FILENAME } else { $args[2] }

        Write-Output "Running tests over $packages and saving output to $coverfilepath"
        go test -race $packages -covermode atomic -coverprofile $coverfilepath
        if(!$?) {
           exit $?
        }
        go tool cover -html="$coverfilepath"
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