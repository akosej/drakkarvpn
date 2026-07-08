package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/tun"
)

type Profile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
	DNS        string `json:"dns"`
	Endpoint   string `json:"endpoint"`
	AllowedIPs string `json:"allowedIPs"`
}

// App struct
type App struct {
	ctx       context.Context
	dev       tun.Device
	ws        *websocket.Conn
	isRunning bool
	mu        sync.Mutex
	profiles  []Profile

	// Métricas
	txBytes  uint64
	rxBytes  uint64
	activeID string
}

func (a *App) GetMetrics() map[string]interface{} {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]interface{}{
		"tx":       a.txBytes,
		"rx":       a.rxBytes,
		"activeID": a.activeID,
	}
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{
		profiles: []Profile{},
	}
	app.loadProfiles()
	return app
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	log.Println("Ejecutando startup de la aplicación...")
	a.ctx = ctx
	log.Println("Startup completado.")
}

func (a *App) loadProfiles() {
	// Obtener la carpeta home del usuario
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error obteniendo home dir: %v", err)
		return
	}

	configDir := filepath.Join(homeDir, ".akosvpn")
	profilePath := filepath.Join(configDir, "profiles.json")

	// Crear el directorio si no existe
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		os.MkdirAll(configDir, 0755)
	}

	data, err := os.ReadFile(profilePath)
	if err == nil {
		json.Unmarshal(data, &a.profiles)
	}
}

func (a *App) saveProfiles() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Error obteniendo home dir: %v", err)
		return
	}

	configDir := filepath.Join(homeDir, ".akosvpn")
	profilePath := filepath.Join(configDir, "profiles.json")

	// Asegurar que el directorio existe antes de guardar
	os.MkdirAll(configDir, 0755)

	data, _ := json.MarshalIndent(a.profiles, "", "  ")
	err = os.WriteFile(profilePath, data, 0644)
	if err != nil {
		log.Printf("Error guardando perfiles en %s: %v", profilePath, err)
	}
}

func (a *App) GetProfiles() []Profile {
	return a.profiles
}

func (a *App) SaveProfile(p Profile) {
	a.mu.Lock()
	defer a.mu.Unlock()

	found := false
	for i, profile := range a.profiles {
		if profile.ID == p.ID {
			a.profiles[i] = p
			found = true
			break
		}
	}
	if !found {
		p.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		a.profiles = append(a.profiles, p)
	}
	a.saveProfiles()
}

func (a *App) DeleteProfile(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	newProfiles := []Profile{}
	for _, p := range a.profiles {
		if p.ID != id {
			newProfiles = append(newProfiles, p)
		}
	}
	a.profiles = newProfiles
	a.saveProfiles()
}

func (a *App) Connect(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isRunning {
		return fmt.Errorf("VPN ya está corriendo")
	}

	var p *Profile
	for _, prof := range a.profiles {
		if prof.ID == id {
			p = &prof
			break
		}
	}
	if p == nil {
		return fmt.Errorf("perfil no encontrado")
	}

	if !isAdmin() {
		return fmt.Errorf("se requiere ejecutar como administrador")
	}

	dev, err := tun.CreateTUN("wg0", 1420)
	if err != nil {
		return fmt.Errorf("error TUN: %v", err)
	}
	a.dev = dev

	ip := strings.Split(p.Address, "/")[0]
	exec.Command("netsh", "interface", "ip", "set", "address", "name=wg0", "static", ip, "255.255.255.0").Run()
	if p.DNS != "" {
		exec.Command("netsh", "interface", "ip", "set", "dns", "name=wg0", "static", p.DNS).Run()
	}

	dialer := websocket.Dialer{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	ws, _, err := dialer.Dial(p.Endpoint, nil)
	if err != nil {
		dev.Close()
		return fmt.Errorf("error WS: %v", err)
	}
	a.ws = ws
	a.isRunning = true
	a.activeID = id
	a.txBytes = 0
	a.rxBytes = 0

	errChan := make(chan error, 2)
	go a.loopSend(errChan)
	go a.loopReceive(errChan)

	go func() {
		err := <-errChan
		log.Printf("VPN Error: %v", err)
		a.Disconnect()
	}()

	return nil
}

func (a *App) Disconnect() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.ws != nil {
		a.ws.Close()
		a.ws = nil
	}
	if a.dev != nil {
		a.dev.Close()
		a.dev = nil
	}
	a.isRunning = false
	a.activeID = ""
}

func (a *App) IsRunning() bool {
	return a.isRunning
}

func (a *App) loopSend(errChan chan error) {
	packet := make([]byte, 2048)
	packets := [][]byte{packet}
	sizes := make([]int, 1)
	for {
		n, err := a.dev.Read(packets, sizes, 0)
		if err != nil {
			errChan <- err
			return
		}
		if n > 0 {
			a.mu.Lock()
			a.txBytes += uint64(sizes[0])
			a.mu.Unlock()
			if err := a.ws.WriteMessage(websocket.BinaryMessage, packet[:sizes[0]]); err != nil {
				errChan <- err
				return
			}
		}
	}
}

func (a *App) loopReceive(errChan chan error) {
	for {
		mType, data, err := a.ws.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}
		if mType == websocket.BinaryMessage {
			a.mu.Lock()
			a.rxBytes += uint64(len(data))
			a.mu.Unlock()
			packets := [][]byte{data}
			if _, err := a.dev.Write(packets, 0); err != nil {
				errChan <- err
				return
			}
		}
	}
}

func isAdmin() bool {
	var sid *windows.SID
	windows.AllocateAndInitializeSid(&windows.SECURITY_NT_AUTHORITY, 2, windows.SECURITY_BUILTIN_DOMAIN_RID, windows.DOMAIN_ALIAS_RID_ADMINS, 0, 0, 0, 0, 0, 0, &sid)
	defer windows.FreeSid(sid)
	token := windows.Token(0)
	member, _ := token.IsMember(sid)
	return member
}
