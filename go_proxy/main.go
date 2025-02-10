package main

import (
	"bufio"
	"go_proxy/Snippets"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	address           = ":6666"
	connectionTimeout = 60 * time.Second
	readWriteTimeout  = 10 * time.Second
)

func main() {

	// Create a listener
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Fehler beim Starten des Servers: %v", err)
	}
	log.Println("Starting proxy server on :6666")

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Fehler beim Akzeptieren der Verbindung: %v", err)
			continue
		}
		// Deadlines setzen
		if err := conn.SetDeadline(time.Now().Add(connectionTimeout)); err != nil { // Beispiel: 10 Sekunden gesamt
			log.Println("Error setting deadline:", err)
			conn.Close() // Wichtig: Verbindung schließen bei Fehler
			return
		}
		// Handle the connection in a new goroutine
		go handleConnection(conn)
	}
}

// --------------------------------------------------------------------------------------------
func handleConnection(conn net.Conn) {
	defer conn.Close()

	var BrowserHeader strings.Builder
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		BrowserHeader.WriteString(line + "\n")
		if line == "" {
			break // End of headers
		}
	}

	if err := scanner.Err(); err != nil {
		log.Println("Error reading request:", err)
		return
	}

	if !Snippets.Income_init(BrowserHeader.String()) {
		return
	}
	sMethod := Snippets.DataIncome.Method

	if sMethod == "CONNECT" {
		httpsWorker(conn, &Snippets.DataIncome)
	} else {
		httpWorker(conn, &Snippets.DataIncome)
	}

}

// --------------------------------------------------------------------------------------------
func httpWorker(conn net.Conn, Data *Snippets.Income) {

	host := Data.Host
	port := Data.Port

	newHeader := CreateHeader(Data)
	if !Snippets.CheckAuth(conn, Data.Header) {
		return
	}

	if port == 0 {
		port = 80
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))

	remoteConn, err := net.DialTimeout("tcp", address, connectionTimeout)
	if err != nil {
		log.Printf("Fehler beim Verbinden mit %s: %v", address, err)
		conn.Close()
		return
	}
	defer remoteConn.Close()
	remoteConn.SetDeadline(time.Now().Add(readWriteTimeout))
	log.Println("---> HTTP Verbindung hergestellt zu", remoteConn.RemoteAddr())

	// Send the request
	_, err = remoteConn.Write([]byte(newHeader))
	if err != nil {
		log.Printf("Fehler beim Senden der Anfrage: %v", err)
		remoteConn.Close()
		return
	}

	// Stream the response back to the client
	_, err = io.Copy(conn, remoteConn)
	if err != nil {
		log.Printf("Fehler beim Kopieren der Antwort: %v", err)
	}
}

// --------------------------------------------------------------------------------------------
func httpsWorker(conn net.Conn, Data *Snippets.Income) {
	defer conn.Close()

	if !Snippets.CheckAuth(conn, Data.Header) {
		return
	}

	// Zieladresse erstellen
	target := net.JoinHostPort(Data.Host, strconv.Itoa(Data.Port))

	// Stelle eine Verbindung zum Ziel her
	targetConn, err := net.DialTimeout("tcp", target, connectionTimeout)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", target, err)
		Snippets.WriteResponse(conn, strconv.Itoa(http.StatusBadGateway), "") // Informiere den Client über den Fehler
		return
	}
	defer targetConn.Close()
	targetConn.SetDeadline(time.Now().Add(connectionTimeout))
	log.Println("---> HTTPS Verbindung hergestellt zu", targetConn.RemoteAddr())

	// Sende die "200 Connection Established"-Antwort
	_, err = conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	if err != nil {
		log.Printf("Failed to send 200 response: %v", err)
		return
	}

	log.Println("Connected to", target)

	// Starte den Tunnel zwischen Client und Server

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, err := io.Copy(targetConn, conn)
		if err != nil {
			log.Printf("Error copying data to target: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		_, err := io.Copy(conn, targetConn)
		if err != nil {
			log.Printf("Error copying data to client: %v", err)
		}
	}()

	wg.Wait()
}

// --------------------------------------------------------------------------------------------
func CreateHeader(Data *Snippets.Income) string {

	request := ""
	parts := strings.Split(Data.Header, "\n")
	for _, val := range parts {
		if strings.Contains(val, "Connection: keep-alive") {
			request += "Connection: close\r\n"
			continue
		}
		request += val + "\r\n"
	}

	return request + "\r\n"
}
