package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/MatiasMarchant/Prueba1/tree/master/chat"
)

//RegistroCamion es la EDD para llevar registro de los paquetes del camión
type RegistroCamion struct {
	idpaquete    string
	tipo         string
	valor        string
	origen       string
	destino      string
	intentos     string
	fechaentrega time.Time
}

//escribirRegistro escribe en un .csv el historial de los paquetes del camión
func escribirRegistro(ListaRegistroCamion []RegistroCamion, idpaquete string, writer *csv.Writer) {
	for _, elem := range ListaRegistroCamion {
		if elem.idpaquete == idpaquete {
			aescribir := []string{
				elem.idpaquete,
				elem.tipo,
				elem.valor,
				elem.origen,
				elem.destino,
				elem.intentos,
				time.Now().String(),
			}
			writer.Write(aescribir)
			writer.Flush()
		}
	}
}

//preguntasinicialescamion es la función que se encarga de pedir input al usuario sobre cuánto tiempo espera el camión por un 2do paquete y cuánto demora en enviarlos
func preguntasinicialescamion() (string, string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Preguntas iniciales")
	fmt.Println("Tiempo en segundos de espera por segundo paquete")
	fmt.Printf("> ")
	tiempoespera, _ := reader.ReadBytes('\n')
	fmt.Println("Tiempo en segundos demora envio paquete")
	fmt.Printf("> ")
	tiempodemora, _ := reader.ReadBytes('\n')
	return string(tiempoespera), string(tiempodemora)
}

//Entregarpaquete es la función que ambos camiones de retail llaman para hacer la entrega de un paquete específico. En esta función se manejan las restricciones de reintentos
// y se hacen RPC's a logistica
func Entregarpaquete(paquete *chat.ColaPaquete, ListaRegistroCamion []RegistroCamion, c chat.ChatServiceClient, idcamion string, intentos string, tiempodemoraint int) chat.ColaPaquete {
	paquete.Estado = "En camino"
	time.Sleep(time.Second * time.Duration(int64(tiempodemoraint)))
	exito := rand.Float64()
	if exito < 0.8 {
		// Se entrega paquete
		// Se modifica registro - fechaentrega
		for _, elem := range ListaRegistroCamion {
			if elem.idpaquete == paquete.Idpaquete {
				elem.fechaentrega = time.Now()
			}
		}
		paquete.Estado = "Recibido"
		// Se notifica a logistica
		nuevoPaqueteEnviado := chat.PaqueteEnviado{
			Idpaquete:   paquete.Idpaquete,
			Seguimiento: paquete.Seguimiento,
			Tipo:        paquete.Tipo,
			Valor:       paquete.Valor,
			Intentos:    paquete.Intentos,
			Estado:      paquete.Estado,
			Origen:      paquete.Origen,
			Destino:     paquete.Destino,
			Idcamion:    idcamion,
		}

		c.ActualizarRegistroPaqueteCamionRetail(context.Background(), &nuevoPaqueteEnviado)

	} else {
		// No se entrega paquete - depende de tipo ver q se hace
		switch paquete.Tipo {
		case "retail":
			//
			contintentos, _ := strconv.Atoi(paquete.Intentos)
			if contintentos < 3 {
				// Dormir, subir intento
				time.Sleep(time.Second * time.Duration(int64(tiempodemoraint)))
				contintentos++
				strcontintentosmasuno := strconv.Itoa(contintentos)
				paquete.Intentos = strcontintentosmasuno
				// Re enviar
				*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion, paquete.Intentos, tiempodemoraint)
			}
		case "prioritario":
			//
			contintentos, _ := strconv.Atoi(paquete.Intentos)

			if contintentos < 2 {
				// Se debe calcular si esq se debe reintentar
				valorint, _ := strconv.Atoi(paquete.Valor)
				costo := float64(10*contintentos) + 0.3*float64(contintentos)*float64(valorint)
				if costo < float64(valorint) {
					// Se reintenta (se duerme y se aumenta intentos)
					time.Sleep(time.Second * time.Duration(int64(tiempodemoraint)))
					contintentos++
					strcontintentosmasuno := strconv.Itoa(contintentos)
					paquete.Intentos = strcontintentosmasuno
					// Re enviar
					*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion, paquete.Intentos, tiempodemoraint)
				}
			} // Si no se debe reintentar, no entra al if y cae mas abajo
		}

		// Si llega acá quiere decir que no se pudo enviar -> "No Recibido"
		paquete.Estado = "No Recibido"
		// Se actualiza timestamp
		for _, elem := range ListaRegistroCamion {
			if elem.idpaquete == paquete.Idpaquete {
				elem.fechaentrega = time.Now()
			}
		}
		// Se notifica a logistica
		nuevoPaqueteEnviado := chat.PaqueteEnviado{
			Idpaquete:   paquete.Idpaquete,
			Seguimiento: paquete.Seguimiento,
			Tipo:        paquete.Tipo,
			Valor:       paquete.Valor,
			Intentos:    paquete.Intentos,
			Estado:      paquete.Estado,
			Origen:      paquete.Origen,
			Destino:     paquete.Destino,
			Idcamion:    idcamion,
		}

		c.ActualizarRegistroPaqueteCamionRetail(context.Background(), &nuevoPaqueteEnviado)
	}

	retPaquete := chat.ColaPaquete{
		Idpaquete:   "9999",
		Seguimiento: "9999",
		Tipo:        "9999",
		Valor:       "9999",
		Intentos:    "9999",
		Estado:      "9999",
		Origen:      "9999",
		Destino:     "9999",
	}
	return retPaquete
}

//Entregarpaquetenormal es la función que el camión normal llama para hacer la entrega de un paquete específico. En esta función se manejan las restricciones de reintentos
// y se hacen RPC's a logistica
func Entregarpaquetenormal(paquete *chat.ColaPaquete, ListaRegistroCamion []RegistroCamion, c chat.ChatServiceClient, idcamion string, intentos string, tiempodemoraint int) chat.ColaPaquete {
	paquete.Estado = "En camino"
	time.Sleep(time.Second * time.Duration(int64(tiempodemoraint)))
	exito := rand.Float64()
	if exito < 0.8 {
		// Se entrega paquete
		// Se modifica registro - fechaentrega
		for _, elem := range ListaRegistroCamion {
			if elem.idpaquete == paquete.Idpaquete {
				elem.fechaentrega = time.Now()
			}
		}
		paquete.Estado = "Recibido"
		// Se notifica a logistica
		nuevoPaqueteEnviado := chat.PaqueteEnviado{
			Idpaquete:   paquete.Idpaquete,
			Seguimiento: paquete.Seguimiento,
			Tipo:        paquete.Tipo,
			Valor:       paquete.Valor,
			Intentos:    paquete.Intentos,
			Estado:      paquete.Estado,
			Origen:      paquete.Origen,
			Destino:     paquete.Destino,
			Idcamion:    idcamion,
		}

		c.ActualizarRegistroPaqueteCamionNormal(context.Background(), &nuevoPaqueteEnviado)

	} else {
		// No se entrega paquete - depende de tipo ver q se hace
		switch paquete.Tipo {
		case "prioritario":
			//
			contintentos, _ := strconv.Atoi(paquete.Intentos)

			if contintentos < 2 {
				// Se debe calcular si esq se debe reintentar
				valorint, _ := strconv.Atoi(paquete.Valor)
				costo := float64(10*contintentos) + 0.3*float64(contintentos)*float64(valorint)
				if costo < float64(valorint) {
					// Se reintenta (se duerme y se aumenta intentos)
					time.Sleep(time.Second * time.Duration(int64(tiempodemoraint)))
					contintentos++
					strcontintentosmasuno := strconv.Itoa(contintentos)
					paquete.Intentos = strcontintentosmasuno
					// Re enviar
					*paquete = Entregarpaquetenormal(paquete, ListaRegistroCamion, c, idcamion, paquete.Intentos, tiempodemoraint)
				}
			} // Si no se debe reintentar, no entra al if y cae mas abajo

		case "normal":
			//
			contintentos, _ := strconv.Atoi(paquete.Intentos)

			if contintentos < 2 {
				// Se debe calcular si esq se debe reintentar
				valorint, _ := strconv.Atoi(paquete.Valor)
				costo := 10 * contintentos
				if costo < valorint {
					// Se reintenta (se duerme y se aumenta intentos)
					time.Sleep(time.Second * time.Duration(int64(tiempodemoraint)))
					contintentos++
					strcontintentosmasuno := strconv.Itoa(contintentos)
					paquete.Intentos = strcontintentosmasuno
					// Re enviar
					*paquete = Entregarpaquetenormal(paquete, ListaRegistroCamion, c, idcamion, paquete.Intentos, tiempodemoraint)
				}
			}
		}

		// Si llega acá quiere decir que no se pudo enviar -> "No Recibido"
		paquete.Estado = "No Recibido"
		// Se actualiza timestamp
		for _, elem := range ListaRegistroCamion {
			if elem.idpaquete == paquete.Idpaquete {
				elem.fechaentrega = time.Now()
			}
		}
		// Se notifica a logistica
		nuevoPaqueteEnviado := chat.PaqueteEnviado{
			Idpaquete:   paquete.Idpaquete,
			Seguimiento: paquete.Seguimiento,
			Tipo:        paquete.Tipo,
			Valor:       paquete.Valor,
			Intentos:    paquete.Intentos,
			Estado:      paquete.Estado,
			Origen:      paquete.Origen,
			Destino:     paquete.Destino,
			Idcamion:    idcamion,
		}

		c.ActualizarRegistroPaqueteCamionNormal(context.Background(), &nuevoPaqueteEnviado)
	}
	return *paquete
}

// ---------------------------------------- CAMION RETAIL2 ------------------------------------------------------------------
//camionretail2 es la función que corre el 2do camión de retail, acá se genera el archivo .csv donde se guardará su registro.
func camionretail2(tiempoespera string, tiempodemora string) {
	csvfile, erres := os.Create("registroretail2.csv")
	if erres != nil {
		log.Fatalf("No pude crear %s", erres)
	}
	csvwriter := csv.NewWriter(csvfile)
	defer csvwriter.Flush()
	primeralinea := []string{
		"id-paquete",
		"tipo",
		"valor",
		"origen",
		"destino",
		"intentos",
		"fecha-entrega",
	}
	csvwriter.Write(primeralinea)
	csvwriter.Flush()

	var ListaRegistroCamion []RegistroCamion
	tiempodemoraint, _ := strconv.Atoi(strings.TrimSuffix(tiempodemora, "\n")) // CUIDADO LINUX
	tiempoesperaint, _ := strconv.Atoi(strings.TrimSuffix(tiempoespera, "\n")) // CUIDADO LINUX

	var conn *grpc.ClientConn
	conn, err := grpc.Dial("dist37:9000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No me pude conectar al puerto 9000: %s", err)
	}
	defer conn.Close()

	c := chat.NewChatServiceClient(conn)

	idcamion := chat.IdCamion{
		Idcamion: "3",
	}

	for true {
		paquete, _ := c.EntregarPaqueteCamionRetail(context.Background(), &idcamion)
		if paquete.Idpaquete == "NoPaquetes" { // Si no encuentra paquetes, dormir
			time.Sleep(time.Second * time.Duration(int64(tiempoesperaint)))
		} else { // Si encontró paquete, dormir para esperar el 2do y si no, marchar
			time.Sleep(time.Second * time.Duration(int64(tiempoesperaint)))
			paquete2, _ := c.EntregarPaqueteCamionRetail(context.Background(), &idcamion)
			if paquete2.Idpaquete == "Nopaquetes" { // Solo paquete

				// Primero se ingresa a su registro
				nuevoRegistro := RegistroCamion{
					idpaquete:    paquete.Idpaquete,
					tipo:         paquete.Tipo,
					valor:        paquete.Valor,
					origen:       paquete.Origen,
					destino:      paquete.Destino,
					intentos:     paquete.Intentos,
					fechaentrega: time.Time{},
				}

				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro)

				// Marchar solo con paquete
				*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
				escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)
			} else {
				// paquete y paquete2

				// Primero se ingresa a su registro
				nuevoRegistro1 := RegistroCamion{
					idpaquete:    paquete.Idpaquete,
					tipo:         paquete.Tipo,
					valor:        paquete.Valor,
					origen:       paquete.Origen,
					destino:      paquete.Destino,
					intentos:     paquete.Intentos,
					fechaentrega: time.Time{},
				}

				nuevoRegistro2 := RegistroCamion{
					idpaquete:    paquete2.Idpaquete,
					tipo:         paquete2.Tipo,
					valor:        paquete2.Valor,
					origen:       paquete2.Origen,
					destino:      paquete2.Destino,
					intentos:     paquete2.Intentos,
					fechaentrega: time.Time{},
				}

				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro1)
				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro2)

				// Marchar con paquete y paquete2 (ver cual es mas caro)
				paquete.Estado = "En camino"
				paquete2.Estado = "En camino"

				valor1, _ := strconv.Atoi(paquete.Valor)
				valor2, _ := strconv.Atoi(paquete2.Valor)

				if valor1 > valor2 {
					// Se entrega paquete primero

					*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)
					*paquete2 = Entregarpaquete(paquete2, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete2.Idpaquete, csvwriter)
				} else {
					// Se entrega paquete2 primero
					*paquete2 = Entregarpaquete(paquete2, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete2.Idpaquete, csvwriter)
					*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)

				}
			}
		}

	}

}

// ---------------------------------------- CAMION RETAIL1 ------------------------------------------------------------------
//camionretail1 es la función que corre el 1er camión de retail, acá se genera el archivo .csv donde se guardará su registro.
func camionretail1(tiempoespera string, tiempodemora string) {
	csvfile, erres := os.Create("registroretail1.csv")
	if erres != nil {
		log.Fatalf("No pude crear %s", erres)
	}
	csvwriter := csv.NewWriter(csvfile)
	defer csvwriter.Flush()
	primeralinea := []string{
		"id-paquete",
		"tipo",
		"valor",
		"origen",
		"destino",
		"intentos",
		"fecha-entrega",
	}
	csvwriter.Write(primeralinea)
	csvwriter.Flush()

	var ListaRegistroCamion []RegistroCamion
	tiempodemoraint, _ := strconv.Atoi(strings.TrimSuffix(tiempodemora, "\n")) // CUIDADO LINUX
	tiempoesperaint, _ := strconv.Atoi(strings.TrimSuffix(tiempoespera, "\n")) // CUIDADO LINUX

	var conn *grpc.ClientConn
	conn, err := grpc.Dial("dist37:9000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No me pude conectar al puerto 9001: %s", err)
	}
	defer conn.Close()

	c := chat.NewChatServiceClient(conn)

	idcamion := chat.IdCamion{
		Idcamion: "2",
	}

	for true {
		paquete, _ := c.EntregarPaqueteCamionRetail(context.Background(), &idcamion)
		if paquete.Idpaquete == "NoPaquetes" { // Si no encuentra paquetes, dormir
			time.Sleep(time.Second * time.Duration(int64(tiempoesperaint)))
		} else { // Si encontró paquete, dormir para esperar el 2do y si no, marchar
			time.Sleep(time.Second * time.Duration(int64(tiempoesperaint)))
			paquete2, _ := c.EntregarPaqueteCamionRetail(context.Background(), &idcamion)
			if paquete2.Idpaquete == "Nopaquetes" { // Solo paquete

				// Primero se ingresa a su registro
				nuevoRegistro := RegistroCamion{
					idpaquete:    paquete.Idpaquete,
					tipo:         paquete.Tipo,
					valor:        paquete.Valor,
					origen:       paquete.Origen,
					destino:      paquete.Destino,
					intentos:     paquete.Intentos,
					fechaentrega: time.Time{},
				}

				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro)

				// Marchar solo con paquete
				*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
				escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)
			} else {
				// paquete y paquete2

				// Primero se ingresa a su registro
				nuevoRegistro1 := RegistroCamion{
					idpaquete:    paquete.Idpaquete,
					tipo:         paquete.Tipo,
					valor:        paquete.Valor,
					origen:       paquete.Origen,
					destino:      paquete.Destino,
					intentos:     paquete.Intentos,
					fechaentrega: time.Time{},
				}

				nuevoRegistro2 := RegistroCamion{
					idpaquete:    paquete2.Idpaquete,
					tipo:         paquete2.Tipo,
					valor:        paquete2.Valor,
					origen:       paquete2.Origen,
					destino:      paquete2.Destino,
					intentos:     paquete2.Intentos,
					fechaentrega: time.Time{},
				}

				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro1)
				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro2)

				// Marchar con paquete y paquete2 (ver cual es mas caro)
				paquete.Estado = "En camino"
				paquete2.Estado = "En camino"

				valor1, _ := strconv.Atoi(paquete.Valor)
				valor2, _ := strconv.Atoi(paquete2.Valor)

				if valor1 > valor2 {
					// Se entrega paquete primero

					*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)
					*paquete2 = Entregarpaquete(paquete2, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete2.Idpaquete, csvwriter)
				} else {
					// Se entrega paquete2 primero
					*paquete2 = Entregarpaquete(paquete2, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete2.Idpaquete, csvwriter)
					*paquete = Entregarpaquete(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)

				}
			}
		}

	}

}

// ---------------------------------------- CAMION NORMAL ------------------------------------------------------------------
//camionnormal es la función que corre el camión normal, acá se genera el archivo .csv donde se guardará su registro.
func camionnormal(tiempoespera string, tiempodemora string) {
	csvfile, erres := os.Create("registrocamionnormal.csv")
	if erres != nil {
		log.Fatalf("No pude crear %s", erres)
	}
	csvwriter := csv.NewWriter(csvfile)
	defer csvwriter.Flush()
	primeralinea := []string{
		"id-paquete",
		"tipo",
		"valor",
		"origen",
		"destino",
		"intentos",
		"fecha-entrega",
	}
	csvwriter.Write(primeralinea)
	csvwriter.Flush()

	var ListaRegistroCamion []RegistroCamion
	tiempodemoraint, _ := strconv.Atoi(strings.TrimSuffix(tiempodemora, "\n")) // CUIDADO LINUX
	tiempoesperaint, _ := strconv.Atoi(strings.TrimSuffix(tiempoespera, "\n")) // CUIDADO LINUX

	var conn *grpc.ClientConn
	conn, err := grpc.Dial("dist37:9000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("No me pude conectar al puerto 9001: %s", err)
	}
	defer conn.Close()

	c := chat.NewChatServiceClient(conn)

	idcamion := chat.IdCamion{
		Idcamion: "1",
	}

	for true {
		paquete, _ := c.EntregarPaqueteCamionNormal(context.Background(), &idcamion)
		if paquete.Idpaquete == "NoPaquetes" { // Si no encuentra paquetes, dormir
			time.Sleep(time.Second * time.Duration(int64(tiempoesperaint)))
		} else { // Si encontró paquete, dormir para esperar el 2do y si no, marchar
			time.Sleep(time.Second * time.Duration(int64(tiempoesperaint)))
			paquete2, _ := c.EntregarPaqueteCamionNormal(context.Background(), &idcamion)
			if paquete2.Idpaquete == "Nopaquetes" { // Solo paquete

				// Primero se ingresa a su registro
				nuevoRegistro := RegistroCamion{
					idpaquete:    paquete.Idpaquete,
					tipo:         paquete.Tipo,
					valor:        paquete.Valor,
					origen:       paquete.Origen,
					destino:      paquete.Destino,
					intentos:     paquete.Intentos,
					fechaentrega: time.Time{},
				}

				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro)

				// Marchar solo con paquete
				*paquete = Entregarpaquetenormal(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
				escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)
			} else {
				// paquete y paquete2

				// Primero se ingresa a su registro
				nuevoRegistro1 := RegistroCamion{
					idpaquete:    paquete.Idpaquete,
					tipo:         paquete.Tipo,
					valor:        paquete.Valor,
					origen:       paquete.Origen,
					destino:      paquete.Destino,
					intentos:     paquete.Intentos,
					fechaentrega: time.Time{},
				}

				nuevoRegistro2 := RegistroCamion{
					idpaquete:    paquete2.Idpaquete,
					tipo:         paquete2.Tipo,
					valor:        paquete2.Valor,
					origen:       paquete2.Origen,
					destino:      paquete2.Destino,
					intentos:     paquete2.Intentos,
					fechaentrega: time.Time{},
				}

				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro1)
				ListaRegistroCamion = append(ListaRegistroCamion, nuevoRegistro2)

				// Marchar con paquete y paquete2 (ver cual es mas caro)
				paquete.Estado = "En camino"
				paquete2.Estado = "En camino"

				valor1, _ := strconv.Atoi(paquete.Valor)
				valor2, _ := strconv.Atoi(paquete2.Valor)

				if valor1 > valor2 {
					// Se entrega paquete primero

					*paquete = Entregarpaquetenormal(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)
					*paquete2 = Entregarpaquetenormal(paquete2, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete2.Idpaquete, csvwriter)
				} else {
					// Se entrega paquete2 primero
					*paquete2 = Entregarpaquetenormal(paquete2, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete2.Idpaquete, csvwriter)
					*paquete = Entregarpaquetenormal(paquete, ListaRegistroCamion, c, idcamion.Idcamion, paquete.Intentos, tiempodemoraint)
					escribirRegistro(ListaRegistroCamion, paquete.Idpaquete, csvwriter)

				}
			}
		}

	}

}

//main ejecuta la función preguntasinicialescamion y le entrega el resultado como parametros a cada go func, y se mantiene en un loop for true para que las go func no mueran
func main() {
	fmt.Println("Corriendo el sistema de camiones...\n")
	var tiempoespera, tiempodemora string
	tiempoespera, tiempodemora = preguntasinicialescamion()
	go camionnormal(tiempoespera, tiempodemora)
	go camionretail1(tiempoespera, tiempodemora)
	go camionretail2(tiempoespera, tiempodemora)
	for true {

	}
}
