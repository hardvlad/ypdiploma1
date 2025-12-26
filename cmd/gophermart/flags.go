package main

import (
	"flag"
	"os"
)

type programFlags struct {
	RunAddress     string
	Dsn            string
	AccrualAddress string
}

func parseFlags() programFlags {

	var flags programFlags

	flag.StringVar(&flags.RunAddress, "a", ":8080", "адрес запуска HTTP-сервера")
	if envRunAddr, ok := os.LookupEnv("BASE_URL"); ok {
		flags.RunAddress = envRunAddr
	}

	flag.StringVar(&flags.Dsn, "d", "", "строка подключения к базе данных")
	if envDsn, ok := os.LookupEnv("DATABASE_DSN"); ok {
		flags.Dsn = envDsn
	}

	flag.StringVar(&flags.AccrualAddress, "r", "", "адрес системы расчёта начислений")
	if envAccrual, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		flags.AccrualAddress = envAccrual
	}

	flag.Parse()

	return flags
}
