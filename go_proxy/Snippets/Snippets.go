package Snippets

import (
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Income struct {
	Method   string
	Host     string
	Protocol string
	Header   string
	HostPath string
	Port     int
}

var DataIncome = Income{
	Method:   "GET",
	Host:     "",
	Protocol: "",
	Header:   "",
	HostPath: "",
	Port:     80,
}

const (
	user = "franz"
	pass = "5$_almless"
)

func Income_init(header string) bool {

	DataIncome.Header = header
	parts := strings.Split(header, "\n")  // Split the header into lines
	firstline := strings.Fields(parts[0]) // Split the first line into words

	if len(firstline) < 3 {
		log.Println("len(firstline) Snippets Invalid request " + parts[0])
		return false
	}

	// Parse the URL
	parsedURL, err := url.Parse(firstline[1])
	if err != nil {
		log.Println("url.Parse Snippets Error parsing URL:", err)
		return false
	}

	// Extract host and port
	cPort := ExtractPort(firstline[1], header)
	DataIncome.Port, err = strconv.Atoi(cPort)
	if err != nil {
		log.Println("Error converting port to integer:", err)
	}

	// Finde Host
	for _, val := range parts {

		if strings.Contains(val, "Host:") {
			DataIncome.Host = strings.Split(val, ":")[1]
			DataIncome.Host = strings.TrimSpace(DataIncome.Host)
			break
		}
	}
	if DataIncome.Host == "" {
		DataIncome.Host = parsedURL.Host
	}

	// Fill the DataIncome struct
	DataIncome.Method = firstline[0]
	DataIncome.HostPath = parsedURL.Path
	DataIncome.Protocol = parsedURL.Scheme
	return true
}

// --------------------------------------------------------------------------------------------
func FindAuth(header string) (string, string) {
	lines := strings.Split(header, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Proxy-Authorization:") {
			auth := strings.TrimSpace(strings.TrimPrefix(line, "Proxy-Authorization:"))
			return "Proxy-Authorization", auth
		}
	}
	return "", ""
}

// --------------------------------------------------------------------------------------------
func Say(c string) {
	log.Println("SAY: " + c)
}

// --------------------------------------------------------------------------------------------
// Funktion zum Extrahieren von Host und Port aus einer URL
func ExtractPort(Url, header string) (port string) {

	teil1 := strings.Split(Url, "&") // Get value's entfernen
	teil2 := strings.ReplaceAll(teil1[0], "http://", "")
	teil3 := strings.ReplaceAll(teil2, "https://", "")
	UrlParts := strings.Split(teil3, ":")

	if len(UrlParts) == 2 {
		port = UrlParts[1]
	}

	if port == "" {
		if strings.HasPrefix(header, "CONNECT") {
			port = "443"

		} else {
			port = "80" // Fallback

		}
	}

	return port
}

// ------------------------------------------------------------
// Function to extract the "Host" header from raw HTTP headers
func ExtractHost(header string) string {
	lines := strings.Split(header, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Host:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Host:"))
		}
	}
	return ""
}

// ------------------------------------------------------------
// Extracts a specific header from the raw HTTP headers.
func ExtractHeader(headers, key string) string {
	lines := strings.Split(headers, "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], key) {
			return parts[1]
		}
	}
	return ""
}

// ------------------------------------------------------------
// Writes an HTTP response to the connection.
func WriteResponse(conn net.Conn, status, extraHeaders string) {

	fmt.Fprintf(conn, "HTTP/1.1 %s %s\r\n", status, http.StatusText(stringToInt(status))) // StatusText für bessere Lesbarkeit
	if extraHeaders != "" {
		fmt.Fprint(conn, extraHeaders+"\r\n")
	}
	fmt.Fprint(conn, "\r\n")
}

// ------------------------------------------------------------
func CheckAuth(conn net.Conn, BrowserHeader string) bool {

	// ProxyAuth
	// Extract Proxy-Authorization header.
	headers := BrowserHeader
	authHeader := ExtractHeader(headers, "Proxy-Authorization")

	if authHeader == "" {
		WriteResponse(conn, strconv.Itoa(http.StatusProxyAuthRequired), `Proxy-Authenticate: Basic realm="Secure Proxy"`)
		return false
	}

	// Decode and validate credentials.
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(strings.TrimSpace(authHeader), "Basic "))
	if err != nil || len(strings.SplitN(string(payload), ":", 2)) != 2 {
		WriteResponse(conn, strconv.Itoa(http.StatusProxyAuthRequired), `Proxy-Authenticate: Basic realm="Secure Proxy"`)
		return false
	}

	pair := strings.SplitN(string(payload), ":", 2)
	username, password := user, pass // Replace with secure storage.
	if strings.TrimSpace(pair[0]) != username || strings.TrimSpace(pair[1]) != password {
		WriteResponse(conn, strconv.Itoa(http.StatusProxyAuthRequired), `Proxy-Authenticate: Basic realm="Secure Proxy"`)
		return false
	}

	// Successfully authenticated. Forward the request to the target server.
	log.Printf("Authenticated request from %s", conn.RemoteAddr())
	return true
}

// ------------------------------------------------------------------------
func stringToInt(s string) int {
	result, err := strconv.Atoi(s)
	if err != nil {
		panic(err) // Panic if conversion fails
	}
	return result
}
