package golongapprunner

import (
	"bytes"
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
)

// i guess it is not needed here...
var NewLineBytes = []byte{byte('\n')}

// used just for making http request. Response is ignored
func GetHttpContents(url string) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{
		Transport: tr,
	}
	r, err := client.Get(url)
	defer func() {
		if r != nil && r.Body != nil {
			r.Body.Close()
		}
	}()

	if err != nil {
		return
	}

	ioutil.ReadAll(r.Body)
}

func StringToArgs(s string) []string {
	res := []string{}

	for len(s) > 0 {
		i := 0
		if s[i] == ' ' {
			i++
		} else {
			protected := s[i] == '"'
			if protected {
				i++
			}
			for i < len(s) && (s[i] != ' ' || protected) {
				if i > 0 && s[i] == '"' && s[i-1] != '\\' {
					i++
					break
				}
				i++
			}
			//println(s, i)
			a := s[0:i]
			if protected {
				a = a[1 : len(a)-1]
				a = strings.Replace(a, "\\\\", "\\", -1)
				a = strings.Replace(a, "\\\"", "\"", -1)
			}
			res = append(res, a)
		}

		if i < len(s) {
			s = s[i:]
		} else {
			break
		}
	}

	return res
}

// TODO it is not a good function, need to handle more special chars, etc.
func EscapeCmdLineParam(v string) string {
	if strings.Contains(v, " ") || strings.Contains(v, "\"") {

		s := strings.Replace(v, "\\", "\\\\", -1)

		switch runtime.GOOS {
		case "windows":
			s = strings.Replace(s, "\"", "\\\"", -1)
		default:
			s = strings.Replace(s, "\"", "\"\"", -1)
		}

		return "\"" + s + "\""
	} else {
		return v
	}
}

func ArgsToString(args []string) string {
	b := ""
	for _, v := range args {
		b += EscapeCmdLineParam(v) + " "
	}
	// cut last space
	if len(b) > 0 {
		b = b[0 : len(b)-1]
	}
	return b
}

// ScanLines is a split function for a Scanner that returns each line of
// text, stripped of any trailing end-of-line marker. The returned line may
// be empty. The end-of-line marker is like regular expression notation `\r\n|\r\n`.
// The last non-empty line of input will be returned even if it has no
// newline.
// Note: drawback is that during some strange bad conditions you'll get \r without \n
// because \n will be given in the next batch of bytes, so use it on your own risk

func ModScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {

	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, dropCR(data[0:i]), nil
	}
	if i := bytes.IndexByte(data, '\r'); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}
