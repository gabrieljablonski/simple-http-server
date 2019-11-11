package main

import (
	"bufio"
	"fmt"
	"github.com/kardianos/osext"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	CRLF = "\r\n"
	DefaultContentPreviewLength = 30
)

func readFile(filePath string) ([]byte, error) {
	basePath, err := osext.ExecutableFolder()
	if err != nil{
		return nil, fmt.Errorf("failed to get base path: %s", err)
	}
	absFilePath := basePath + string(os.PathSeparator) + filePath
	file, err := os.Open(absFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s", err)
	}
	return ioutil.ReadAll(file)
}

func prefixWithHost(message string, host string) string {
	return fmt.Sprintf("%s >> %s", host, message)
}

func buildHTML(title, body string) []byte {
	html := fmt.Sprintf(`
	<html>
		<head>
			<title>%s</title>
		</head>
		<body>
			<h1>%s</h1>
			%s
		</body>
	</html>
	`, title, title, body)
	return []byte(strings.Trim(html, "\n\t") + CRLF)
}

func getContentType(filePath string) string {
	extension := strings.Split(filePath, ".")
	switch extension[len(extension)-1] {
	default:
		return "application/octet-stream"
	case "htm", "html":
		return "text/html"
	case "ram", "ra":
		return "audio/x-pn-realaudio"
	case "jpg", "jpeg":
		return "image/jpeg"
	}
}

func buildHTTPHeader(statusCode int, httpVersion, contentType string) []byte {
	var status string
	switch statusCode {
	default:
		status = "Status Code Not Implemented"
	case 200:
		status = "OK"
	case 404:
		status = "File Not Found"
	case 500:
		status = "Internal Server Error"
	}

	statusHeader := fmt.Sprintf("HTTP/%s %d %s", httpVersion, statusCode, status)
	contentTypeHeader := fmt.Sprintf("Content-Type: %s", contentType)
	httpHeader := []string{
		statusHeader,
		contentTypeHeader,
	}
	return []byte(strings.Join(httpHeader, CRLF) + CRLF)
}

func handleConnection(c net.Conn) {
	var msg string
	remoteAddr := c.RemoteAddr().String()
	log.Printf(prefixWithHost("Serving.", remoteAddr))

	var responseHeader []byte
	var responseBody []byte
	for {
		request, err := bufio.NewReader(c).ReadString('\n')
		if err != nil {
			msg = fmt.Sprintf("Failed to read incoming request: %s", err)
			log.Printf(prefixWithHost(msg, remoteAddr))
			break
		}

		msg = fmt.Sprintf("Incoming request: %#v.", request)
		log.Printf(prefixWithHost(msg, remoteAddr))

		re := regexp.MustCompile(
			`GET /((?P<file_name>\w+)\.(?P<file_ext>\w+))? HTTP/(?P<http_version>\d\.\d)\r\n`,
		)

		matches := re.FindAllStringSubmatch(request, -1)
		if matches == nil {
			responseHeader = buildHTTPHeader(404, "1.1", "text/html")
			responseBody = buildHTML("404 Not Found", "Page not found")

			log.Printf(prefixWithHost("Failed to parse request.", remoteAddr))
			break
		}
		var filePath string
		match := matches[0]
		endpoint := match[1]
		if endpoint == "" {
			filePath = "index.html"
		} else {
			filePath = endpoint
		}
		httpVersion := match[4]

		msg = fmt.Sprintf("Reading file: %s", filePath)
		log.Printf(prefixWithHost(msg, remoteAddr))
		content, err := readFile(filePath)
		if err != nil {
			responseHeader = buildHTTPHeader(404, httpVersion, "text/html")
			responseBody = buildHTML("404 Not Found", "File not found")
			msg = fmt.Sprintf("Error reading file: %s", err)
			log.Printf(prefixWithHost(msg, remoteAddr))
			break
		}

		msg = fmt.Sprintf("File content preview: %#v...", string(content)[:DefaultContentPreviewLength])
		log.Printf(prefixWithHost(msg, remoteAddr))

		responseHeader = buildHTTPHeader(200, httpVersion, getContentType(filePath))
		responseBody = content
		break
	}

	if responseHeader != nil {
		msg = fmt.Sprintf("Sending response: %#v.", string(responseHeader))
		if responseBody != nil {
			msg += fmt.Sprintf("\nContent preview: %#v...", string(responseBody[:DefaultContentPreviewLength]))
		}
		log.Printf(prefixWithHost(msg, remoteAddr))
		// append line break to indicate header ending
		response := append(responseHeader, CRLF...)
		response = append(response, responseBody...)
		_, err := c.Write(response)
		if err != nil {
			msg = fmt.Sprintf("Connection lost: %s.", err)
			log.Fatalf(prefixWithHost(msg, remoteAddr))
		}
	}

	err := c.Close()
	if err != nil {
		msg = fmt.Sprintf("Failed closing connection: %s.", err)
		log.Fatalf(prefixWithHost(msg, remoteAddr))
	}
}

func listen(port string) {
	port = ":" + port
	listener, err := net.Listen("tcp4", port)

	if err != nil {
		log.Fatalf("Failed to create listener: %s", err)
	}

	log.Printf("Listening for connections on port %s...\n", port[1:])
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Failed accepting connection: %s", err)
		}
		go handleConnection(conn)
	}
}

func main() {
	arguments := os.Args
	if len(arguments) != 2 {
		filename := arguments[0]
		filename = strings.ReplaceAll(filename, "\\", "/")
		filename = path.Base(filename)
		fmt.Printf("%s usage: %s <port>\n", filename, filename)
		return
	}
	port := arguments[1]
	listen(port)
}
