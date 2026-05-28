package main

import (
    "encoding/json"
    "flag"
    "html/template"
    "log"
    "net/http"
    "strconv"
    "strings"
    "sync"
)

var serialPortName string

type hdmiPort struct {
    Number int    `json:"number"`
    Name   string `json:"name"`
    Active bool   `json:"active"`
}

type server struct {
    templates *template.Template
    mu        sync.RWMutex
    active    int
}

type pageData struct {
    Title       string
    Ports       []hdmiPort
    Selected    int
    Message     string
    MessageType string
}

type jsonResponse struct {
    OK       bool       `json:"ok"`
    Selected int        `json:"selected"`
    Message  string     `json:"message,omitempty"`
    Ports    []hdmiPort `json:"ports,omitempty"`
}

func initSerial(portName string) error {
    if err := openPort(portName); err != nil {
        return err
    }

    if err := switchToPort(1); err != nil {
        return err
    }

    return nil
}

func initWeb() *http.ServeMux {
    templates := template.Must(template.ParseGlob("http/templates/*.html"))

    app := &server{
        templates: templates,
        active:    1,
    }

    mux := http.NewServeMux()
    mux.HandleFunc("/", app.handleIndex)
    mux.HandleFunc("/select", app.handleSelect)
    return mux
}

func main() {
    flag.StringVar(&serialPortName, "port", "/dev/ttyUSB0", "serial port to use")
    flag.Parse()

    if err := initSerial(serialPortName); err != nil {
        log.Fatalf("error initializing serial port: %v\n", err)
    }
    defer closePort()

    mux := initWeb()
    log.Println("HDMI switch web interface listening on http://localhost:8080")
    if err := http.ListenAndServe(":8080", mux); err != nil {
        log.Fatal(err)
    }
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    data := pageData{
        Title:    "HDMI Switch",
        Ports:    s.ports(),
        Selected: s.activePort(),
    }
    s.respond(w, r, http.StatusOK, data, true)
}

func (s *server) handleSelect(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if err := r.ParseForm(); err != nil {
        http.Error(w, "invalid form submission", http.StatusBadRequest)
        return
    }

    portNumber, err := strconv.Atoi(r.FormValue("port"))
    if err != nil || portNumber < 1 || portNumber > 4 {
        s.respond(w, r, http.StatusBadRequest, pageData{
            Title:       "HDMI Switch",
            Ports:       s.ports(),
            Selected:    s.activePort(),
            Message:     "Choose a valid HDMI port.",
            MessageType: "danger",
        }, false)
        return
    }

    if err := switchToPort(portNumber); err != nil {
        s.respond(w, r, http.StatusInternalServerError, pageData{
            Title:       "HDMI Switch",
            Ports:       s.ports(),
            Selected:    s.activePort(),
            Message:     "Unable to switch HDMI input: " + err.Error(),
            MessageType: "danger",
        }, false)
        return
    }

    s.mu.Lock()
    s.active = portNumber
    s.mu.Unlock()
    s.respond(w, r, http.StatusOK, pageData{
        Title:       "HDMI Switch",
        Ports:       s.ports(),
        Selected:    portNumber,
        Message:     "HDMI input switched.",
        MessageType: "success",
    }, true)
}

func (s *server) respond(w http.ResponseWriter, r *http.Request, status int, data pageData, ok bool) {
    if isCurl(r) {
        s.renderJSON(w, status, jsonResponse{
            OK:       ok,
            Selected: data.Selected,
            Message:  data.Message,
            Ports:    data.Ports,
        })
        return
    }

    s.render(w, status, data)
}

func (s *server) render(w http.ResponseWriter, status int, data pageData) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    if err := s.templates.ExecuteTemplate(w, "base", data); err != nil {
        log.Printf("render template: %v", err)
    }
}

func (s *server) renderJSON(w http.ResponseWriter, status int, data jsonResponse) {
    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(data); err != nil {
        log.Printf("render json: %v", err)
    }
}

func isCurl(r *http.Request) bool {
    return strings.HasPrefix(strings.ToLower(r.UserAgent()), "curl/")
}

func (s *server) activePort() int {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.active
}

func (s *server) ports() []hdmiPort {
    active := s.activePort()
    ports := make([]hdmiPort, 0, 4)
    for i := 1; i <= 4; i++ {
        ports = append(ports, hdmiPort{
            Number: i,
            Name:   "HDMI " + strconv.Itoa(i),
            Active: i == active,
        })
    }
    return ports
}
