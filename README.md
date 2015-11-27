### Website Searcher

Given a list of urls in urls.txt, this program fetches each page and determines whether a search term exists on the page.

### Installation Instructions

1. `git clone` this repository
2. `cd` into the directory: `cd go-search`
3. run the `go-search` executable with flags:
	- `-search=searchTerm` 
	- optional flag `-input` specifying the location of the urls file (the default is `urls.txt` in the current working directory)
	- optional flag `-verbose` enables verbose logging

#### Additional Information

- The urls file must be a CSV file with urls in the second column
- The output will be in `results.txt`
