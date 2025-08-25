package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type LogLevel int

const (
	Info LogLevel = iota
	Error
	Warn
	Debug
	Trace
	Route
	Headers
	Body
	Response
	NotFound
	Queries
	Mongo
	MongoError
	Redis
	Database
	Cache
)

func (l LogLevel) String() string {
	return [...]string{
		"INFO", "ERROR", "WARN", "DEBUG", "TRACE", "ROUTE", "HEADERS",
		"BODY", "RESPONSE", "404", "QUERIES", "MONGO", "MONGO_ERROR", "REDIS",
		"DATABASE", "CACHE",
	}[l]
}

func (l LogLevel) color() string {
	return [...]string{
		"\x1b[32m",       // Green
		"\x1b[31m",       // Red
		"\x1b[33m",       // Yellow
		"\x1b[36m",       // Cyan
		"\x1b[35m",       // Magenta
		"\x1b[34m",       // Blue
		"\x1b[36m",       // Cyan
		"\x1b[33m",       // Yellow
		"\x1b[35m",       // Magenta
		"\x1b[38;5;202m", // Reddish Orange
		"\x1b[38;5;93m",  // Purple
		"\x1b[32m",       // Green
		"\x1b[31m",       // Red
		"\x1b[31m",       // Red
		"\x1b[34m",       // Blue
		"\x1b[33m",       // Yellow
	}[l]
}

type DateFormat string

const (
	HourMinute   DateFormat = "hour-minute"
	FullDateTime DateFormat = "full"
)

func getDateFormat() DateFormat {
	format := os.Getenv("LOG_DATE_FORMAT")
	switch format {
	case "full":
		return FullDateTime
	case "hour", "hour-minute":
		return HourMinute
	default:
		return HourMinute
	}
}

func getFormattedTimestamp() string {
	format := getDateFormat()
	now := time.Now()
	switch format {
	case HourMinute:
		return now.Format("15:04:05")
	case FullDateTime:
		return now.Format("02-01-2006 15:04:05")
	default:
		return now.Format("15:04:05")
	}
}

type logMessage struct {
	level   LogLevel
	message string
}

var logChannel = make(chan logMessage, 1000)

func init() {
	ClearConsole()
	PrintBanner()
	go logWorker()
}

func PrintBanner() {
	green := "\x1b[32m"
	reset := "\x1b[0m"
	fmt.Println()
	fmt.Printf("%s  ooooooo                                      o8                           %s\n", green, reset)
	fmt.Printf("%so888   888o oooo  oooo   ooooooo   oo oooooo o888oo oooo   oooo oooo   oooo %s\n", green, reset)
	fmt.Printf("%s888     888  888   888   ooooo888   888   888 888    888   888    888o888   %s\n", green, reset)
	fmt.Printf("%s888o  8o888  888   888 888    888   888   888 888     888 888     o88 88o   %s\n", green, reset)
	fmt.Printf("%s  88ooo88     888o88 8o 88ooo88 8o o888o o888o 888o     8888    o88o   o88o %s\n", green, reset)
	fmt.Printf("%s       88o8                                          o8o888                 %s\n", green, reset)
	fmt.Println()
}

func ClearConsole() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func logWorker() {
	for msg := range logChannel {
		writeLog(msg.level, msg.message)
	}
}

func writeLog(level LogLevel, message string) {
	timestamp := getFormattedTimestamp()
	color := level.color()
	tag := level.String()

	// Handle multi-line messages (like JSON responses) by putting diamond at the end
	if strings.Contains(message, "\n") {
		lines := strings.Split(message, "\n")

		// Find the last non-empty line to add diamond to
		lastNonEmptyIndex := -1
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.TrimSpace(lines[i]) != "" {
				lastNonEmptyIndex = i
				break
			}
		}

		// Print first line without diamond
		fmt.Fprintf(os.Stdout, "\x1b[90m%s\x1b[0m %s[%s]\x1b[0m %s\n", timestamp, color, tag, lines[0])

		// Print remaining lines
		for i := 1; i < len(lines); i++ {
			if i == lastNonEmptyIndex && strings.TrimSpace(lines[i]) != "" {
				// Add diamond to the last non-empty line
				fmt.Fprintf(os.Stdout, "%s %s◆\x1b[0m\n", lines[i], color)
			} else {
				fmt.Fprintf(os.Stdout, "%s\n", lines[i])
			}
		}
	} else {
		// Single line message - use original format
		fmt.Fprintf(os.Stdout, "\x1b[90m%s\x1b[0m %s[%s]\x1b[0m %s %s◆\x1b[0m\n", timestamp, color, tag, message, color)
	}
}

func Log(level LogLevel, message string) {
	select {
	case logChannel <- logMessage{level: level, message: message}:
	default:
		// Channel is full, fallback to synchronous logging
		fmt.Fprintln(os.Stderr, "Async logging channel full. Falling back to sync logging.")
		writeLog(level, message)
	}
}

func LogInfo(message string)     { Log(Info, message) }
func LogError(message string)    { Log(Error, message) }
func LogWarn(message string)     { Log(Warn, message) }
func LogDebug(message string)    { Log(Debug, message) }
func LogTrace(message string)    { Log(Trace, message) }
func LogRoute(message string)    { Log(Route, message) }
func LogHeaders(message string)  { Log(Headers, message) }
func LogBody(message string)     { Log(Body, message) }
func LogQueries(message string)  { Log(Queries, message) }
func LogResponse(message string) { Log(Response, message) }
func LogNotFound(message string) { Log(NotFound, message) }

func LogInfoSync(message string)  { writeLog(Info, message) }
func LogErrorSync(message string) { writeLog(Error, message) }
func LogWarnSync(message string)  { writeLog(Warn, message) }

// MongoDB logging functions
func LogMongo(message string)      { Log(Mongo, message) }
func LogMongoError(message string) { Log(MongoError, message) }

// MongoDB synchronous logging functions
func LogMongoSync(message string)      { writeLog(Mongo, message) }
func LogMongoErrorSync(message string) { writeLog(MongoError, message) }

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture start time immediately
		requestStart := time.Now()

		// Read the body
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(r.Body)
		}
		// Restore the body
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		// Always skip logging for swagger requests
		if strings.HasPrefix(r.URL.Path, "/swagger") {
			next.ServeHTTP(w, r)
			return
		}

		// Log request info IMMEDIATELY (before processing)
		if os.Getenv("LOG_ROUTE") == "true" {
			fmt.Println() // Empty line before route log
			LogRoute(fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		}

		if os.Getenv("LOG_QUERIES") == "true" {
			if query := r.URL.RawQuery; query != "" {
				LogQueries(strings.ReplaceAll(query, "&", ", "))
			}
		}

		if os.Getenv("LOG_HEADERS") == "true" {
			var headerStr strings.Builder
			for key, value := range r.Header {
				headerStr.WriteString(fmt.Sprintf("%s: %s, ", key, strings.Join(value, ",")))
			}
			if headerStr.Len() > 0 {
				LogHeaders(strings.TrimSuffix(headerStr.String(), ", "))
			}
		}

		if os.Getenv("LOG_BODY") == "true" && len(bodyBytes) > 0 {
			LogBody(prettyPrintJSON(bodyBytes))
		}

		lrw := &loggingResponseWriter{w, http.StatusOK, make([]byte, 0)}
		next.ServeHTTP(lrw, r)

		if lrw.statusCode == http.StatusNotFound {
			// The notFoundHandler will log this, so we don't need to do anything here.
			return
		}

		// Calculate elapsed time using time.Since for better precision
		elapsed := time.Since(requestStart)

		responseBody := string(lrw.body)
		if responseBody == "" {
			responseBody = fmt.Sprintf("Status: %d", lrw.statusCode)
		} else {
			// Format JSON responses for better readability
			responseBody = prettyPrintJSON(lrw.body)
		}

		// Format timing based on elapsed duration
		var timingStr string
		if elapsed >= time.Millisecond {
			timingStr = fmt.Sprintf("%.2fms", float64(elapsed.Nanoseconds())/1000000.0)
		} else if elapsed >= time.Microsecond {
			timingStr = fmt.Sprintf("%.2fµs", float64(elapsed.Nanoseconds())/1000.0)
		} else {
			// For very fast operations, show at least 0.01µs to indicate it's not zero
			if elapsed == 0 {
				timingStr = "<0.01µs"
			} else {
				timingStr = fmt.Sprintf("%dns", elapsed.Nanoseconds())
			}
		}

		// Log response AFTER processing (with timing) - only if enabled
		if os.Getenv("LOG_RESPONSE") == "true" {
			LogResponse(fmt.Sprintf("%s - %s - %s", timingStr, getColoredStatus(lrw.statusCode), responseBody))
		}
	})
}

func getColoredStatus(statusCode int) string {
	var color string
	var statusText string

	switch {
	case statusCode >= 200 && statusCode < 300:
		color = "\x1b[32m" // Green for 2xx
		statusText = "✓"
	case statusCode >= 300 && statusCode < 400:
		color = "\x1b[36m" // Cyan for 3xx
		statusText = "→"
	case statusCode >= 400 && statusCode < 500:
		color = "\x1b[31m" // Red for 4xx
		statusText = "⚠"
	case statusCode >= 500:
		color = "\x1b[38;5;160m" // Medium crimson red for 5xx
		statusText = "✗"
	default:
		color = "\x1b[37m" // White for unknown
		statusText = "?"
	}

	reset := "\x1b[0m"
	return fmt.Sprintf("%s%s %d%s", color, statusText, statusCode, reset)
}

func prettyPrintJSON(b []byte) string {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	if err != nil {
		return string(b)
	}
	return out.String()
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(data []byte) (int, error) {
	lrw.body = append(lrw.body, data...)
	return lrw.ResponseWriter.Write(data)
}
