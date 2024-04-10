package main

import (
	"log/slog"
	"net"
	"os"

	"github.com/lmittmann/tint"
	"gitlab.com/neuland-homeland/honeypot/packages/dbip"
	"gitlab.com/neuland-homeland/honeypot/packages/honeypot"
	"gitlab.com/neuland-homeland/honeypot/packages/pipeline"
	"gitlab.com/neuland-homeland/honeypot/packages/set"
	"gitlab.com/neuland-homeland/honeypot/packages/store"
	"gitlab.com/neuland-homeland/honeypot/packages/transport"
)

func main() {
	InitLogger()
	postgresqlDB := store.PostgreSQL{
		Host:     "localhost",
		Port:     5432,
		User:     "postgres",
		Password: "123",
		DBName:   "honeypot",
	}
	httpHoneypot := honeypot.NewHTTP(honeypot.HTTPConfig{
		Port: 80,
	})

	err := httpHoneypot.Start()
	if err != nil {
		panic(err)
	}
	/*
		postgresHoneypot := honeypot.NewPostgres(honeypot.PostgresConfig{
			Port: 5555,
		})

		err = postgresHoneypot.Start()
		if err != nil {
			panic(err)
		}
	*/
	sshHoneypot := honeypot.NewSSH(honeypot.SSHConfig{
		Port: 2027,
	})

	err = sshHoneypot.Start()
	if err != nil {
		panic(err)
	}

	tcpHoneypot := honeypot.NewTCP(honeypot.MostUsedTCPPorts())
	udpHoneypot := honeypot.NewUDP(honeypot.MostUsedUDPPorts())

	err = tcpHoneypot.Start()
	if err != nil {
		panic(err)
	}

	err = udpHoneypot.Start()
	if err != nil {
		panic(err)
	}
	/*
		lifoStore := store.NewTimeLifo[set.Token](time.Duration(24 * time.Hour))
		// create a file decorator to persist the data
		file, err := os.OpenFile("events.log", os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			panic(err)
		}

		fileStore := store.NewFileDecorator[set.Token](
			file,
			store.NewJSONSerializer[set.Token](),
			lifoStore,
		)

		httpTransport := transport.NewHTTP(transport.HTTPConfig{
			Port: 1112,
			// initializes the http transport with the lifo store
			Store: fileStore,
		})
	*/
	socketioTransport := transport.NewSocketIO(transport.SocketIOConfig{
		Port: 1113,
	})
	//httpChan := httpTransport.Listen()
	socketioChan := socketioTransport.Listen()

	dbIp := dbip.NewIpToCountry("dbip-country.csv")
	//get the port of sshHoneypot and pass it to the dbStore

	dbChan := postgresqlDB.Listen()

	// listen for SET events
	setChannel := pipeline.Map(pipeline.Merge(sshHoneypot.GetSETChannel(), tcpHoneypot.GetSETChannel(), udpHoneypot.GetSETChannel()), func(input set.Token) (set.Token, error) {
		input.COUNTRY = dbIp.Lookup(net.ParseIP(input.SUB))
		return input, nil
	})
	//postgresqlDB.DBStore(setChannel)
	pipeline.Broadcast(setChannel, socketioChan, dbChan)
	forever := make(chan bool)
	<-forever
}

func InitLogger() {
	loggingHandler := tint.NewHandler(os.Stdout, &tint.Options{
		AddSource: true,
		Level:     slog.LevelDebug,
	})
	logger := slog.New(loggingHandler)
	slog.SetDefault(logger)
}
