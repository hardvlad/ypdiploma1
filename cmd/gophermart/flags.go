// Получение аргументов запуска сервиса
package main

import (
	"flag"
	"os"
)

// programFlags определяет структуру для хранения аргументов сервиса
// RunAddress - адрес, на котором запускается HTTP сервер
// Dsn - строка подключения к базе данных
// AccrualAddress - адрес системы расчёта начислений
type programFlags struct {
	RunAddress     string
	Dsn            string
	AccrualAddress string
}

// Функция parseFlags парсит аргументы командной строки и переменные окружения
// и возвращает структуру programFlags с соответствующими значениями
func parseFlags() programFlags {

	var flags programFlags

	// получение адреса, на котором запускается HTTP сервер или из аргумента командной строки -a
	// или из переменной окружения BASE_URL
	flag.StringVar(&flags.RunAddress, "a", ":8080", "адрес запуска HTTP-сервера")
	if envRunAddr, ok := os.LookupEnv("BASE_URL"); ok {
		flags.RunAddress = envRunAddr
	}

	// получение строки подключения к базе данных из аргумента командной строки -d
	// или из переменной окружения DATABASE_URI
	flag.StringVar(&flags.Dsn, "d", "", "строка подключения к базе данных")
	if envDsn, ok := os.LookupEnv("DATABASE_URI"); ok {
		flags.Dsn = envDsn
	}

	// получение адреса системы расчёта начислений из аргумента командной строки -r
	// или из переменной окружения ACCRUAL_SYSTEM_ADDRESS
	flag.StringVar(&flags.AccrualAddress, "r", "", "адрес системы расчёта начислений")
	if envAccrual, ok := os.LookupEnv("ACCRUAL_SYSTEM_ADDRESS"); ok {
		flags.AccrualAddress = envAccrual
	}

	flag.Parse()

	return flags
}
