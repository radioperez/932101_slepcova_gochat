package main

import (
	"flag"
	"net"
	"fmt"
	"bufio"
	"os"
	"sync"
	"strings"
)

var (
	clients = make(map[string]net.Conn)
    clientsMu sync.Mutex
)

func main() {
    mode := flag.String("mode", "server", "Specify 'server' or 'client' mode")
    flag.Parse()

    if *mode == "server" {
        startServer()
    } else if *mode == "client" {
        startClient()
    } else {
        fmt.Println("Invalid mode. Use 'server' or 'client'.")
    }
}

// Запуск сервера через TCP.
// listener ждет подключений, и для каждого запускает собственную горутину
func startServer() {
	listener, err := net.Listen("tcp", ":9090")
    if err != nil {
        fmt.Println("Ошибка в запуске сервера:", err)
        return
    }
    defer listener.Close()
    fmt.Println("Сервер запущен на :9090")

	for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Соединение отвержено :(", err)
            continue
        }
        go handleConnection(conn)
    }
}

// Обработка подключений, сначала принятие юзернейма и добавление его в список пользователей,
// Затем ждем сообщений и передаем их в броадкаст
func handleConnection(conn net.Conn) {
	defer conn.Close()

	_, err := conn.Write([]byte("Подключаемся! Введите юзернейм\n"))
	if err != nil {
		fmt.Println("Ошибка при отправке промпта пользователю:", err)
		return
	}

	var reader = bufio.NewReader(conn)
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Ошибка при чтении юзернейма пользователя:", err)
		return
	}
    username = strings.TrimSuffix(username, "\n") // Убираем newline
	
	// !!! Мьютекс для параллельной обработки
	clientsMu.Lock()
	clients[username] = conn
	length := len(clients)
    clientsMu.Unlock()
	connectedMessage := "Подключается пользователь " + username + "\n"
	fmt.Println(connectedMessage)
	broadcastMessage("SYSTEM", connectedMessage)

	_, err = conn.Write([]byte("Добро пожаловать! Помимо Вас подключено " + fmt.Sprint(length-1) + " пользователей.\n"))
	if err != nil {
		fmt.Println("Ошибка при отправке промпта пользователю:", err)
		return
	}

	for {
		message, err := reader.ReadString('\n')
        if err != nil {
			disconnectMessage := username + " отключается.\n"
            fmt.Println(disconnectMessage)
			broadcastMessage("SYSTEM", disconnectMessage)
            clientsMu.Lock()
			delete(clients, username)
            clientsMu.Unlock()
            return
        }

        broadcastMessage(username, message)
	}
}

func broadcastMessage(username string, message string) {
	clientsMu.Lock()
    defer clientsMu.Unlock()

	// Ищем ссылку на пользователя на первом месте в сообщении
	word := strings.Split(message, " ")[0]
	receiver := "all"
	if strings.HasPrefix(word, "@") {
		receiver = word[1:] // Убираем знак @
		conn, exists := clients[receiver]
		if !exists {
			clients[username].Write([]byte("Пользователь "+receiver+" сейчас не доступен\n"))
			return
		}
		messagePrompt := "\r" + username + fmt.Sprint(">") + message
		conn.Write([]byte(messagePrompt))
	} else {
		for receiver, conn := range clients {
			if receiver != username {
				messagePrompt := "\r" + username + fmt.Sprint(">>") + message 
				conn.Write([]byte(messagePrompt))
			}
		}
	}

	fmt.Println(username, "@", receiver, ":", message)
}

func startClient() {
	fmt.Println("Режим клиента")
	conn, err := net.Dial("tcp", "localhost:9090")
    if err != nil {
        fmt.Println("Ошибка при подключении к серверу:", err)
        return
    }
    defer conn.Close()

	go listenForMessages(conn)

	userio := bufio.NewReader(os.Stdin)
	for {
		message, err := userio.ReadString('\n')
		if err != nil {
			fmt.Println("Ошибка чтения ввода: ", err)
			return
		}

		if message == "exit\n" {
			break
		}

		conn.Write([]byte(message))
		printPrompt()
	}
}

func listenForMessages(conn net.Conn) {
	reader := bufio.NewReader(conn)
    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Disconnected from server.")
            return
        }

		message = strings.TrimSuffix(message, "\n")
		fmt.Println(message)
		printPrompt()
    }
}

func printPrompt() {
	fmt.Print(">")
}