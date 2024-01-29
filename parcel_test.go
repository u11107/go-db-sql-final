package main

import (
	"database/sql"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

var store ParcelStore

func assertParcelsEqual(t *testing.T, expected, actual Parcel) {
	assert.Equal(t, expected.Client, actual.Client, "Client values do not match")
	assert.Equal(t, expected.Status, actual.Status, "Status values do not match")
	assert.Equal(t, expected.Address, actual.Address, "Address values do not match")
	assert.Equal(t, expected.CreatedAt, actual.CreatedAt, "CreatedAt values do not match")
}

func TestMain(m *testing.M) {
	db, err := sql.Open("sqlite", "tracker.db")
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

	store = NewParcelStore(db)
	runTests := m.Run()
	os.Exit(runTests)
}

// getTestParcel возвращает тестовую посылку
func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	// prepare
	parcel := getTestParcel()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	number, err := store.Add(parcel)
	require.NoError(t, err, "add error")
	require.NotEqual(t, number, 0, "expected not 0 id")

	// get
	// получите только что добавленную посылку, убедитесь в отсутствии ошибки
	fromDb, err := store.Get(number)
	require.NoError(t, err, "get error ")
	assertParcelsEqual(t, parcel, fromDb)

	// delete
	// удалите добавленную посылку, убедитесь в отсутствии ошибки
	// проверьте, что посылку больше нельзя получить из БД
	err = store.Delete(number)
	require.NoError(t, err, "delete error")

	_, err = store.Get(number)
	require.ErrorIs(t, err, sql.ErrNoRows, "error no rows expected")
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	// prepare
	parcel := getTestParcel()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	number, err := store.Add(parcel)
	require.NoError(t, err, "add error")
	require.NotEqual(t, number, 0, "expected not 0 id")

	// set address
	// обновите адрес, убедитесь в отсутствии ошибки
	newAddress := "new test address"
	err = store.SetAddress(number, newAddress)
	require.NoError(t, err, "update error")

	// check
	// получите добавленную посылку и убедитесь, что адрес обновился
	p, err := store.Get(number)
	require.NoError(t, err, "get error")
	assert.Equal(t, newAddress, p.Address, "expected %s got %s", newAddress, p.Address)
}

// TestSetStatus проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	// prepare
	parcel := getTestParcel()

	// add
	// добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
	number, err := store.Add(parcel)
	require.NoError(t, err, "add error")
	require.NotEqual(t, number, 0, "expected not 0 id")

	// set status
	// обновите статус, убедитесь в отсутствии ошибки
	err = store.SetStatus(number, ParcelStatusSent)
	require.NoError(t, err, "set error")

	// check
	// получите добавленную посылку и убедитесь, что статус обновился
	p, err := store.Get(number)
	require.NoError(t, err, "get error")
	assert.Equal(t, ParcelStatusSent, p.Status, "expected %s got %s", ParcelStatusSent, p.Status)
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// prepare
	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}

	// задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	parcels[0].Client = client
	parcels[1].Client = client
	parcels[2].Client = client

	// add
	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i]) // добавьте новую посылку в БД, убедитесь в отсутствии ошибки и наличии идентификатора
		require.NoError(t, err, "add error")
		require.NotEqual(t, id, 0, "expected id not 0")

		// обновляем идентификатор добавленной у посылки
		parcels[i].Number = id

		// сохраняем добавленную посылку в структуру map, чтобы её можно было легко достать по идентификатору посылки
		parcelMap[id] = parcels[i]
	}

	// get by client
	storedParcels, err := store.GetByClient(client) // получите список посылок по идентификатору клиента, сохранённого в переменной client
	// убедитесь в отсутствии ошибки
	// убедитесь, что количество полученных посылок совпадает с количеством добавленных
	require.NoError(t, err, "get error")
	require.Equal(t, len(parcels), len(storedParcels), "expected result length %d got %d", len(parcels), len(storedParcels))

	// check
	for _, parcel := range storedParcels {
		// в parcelMap лежат добавленные посылки, ключ - идентификатор посылки, значение - сама посылка
		// убедитесь, что все посылки из storedParcels есть в parcelMap
		// убедитесь, что значения полей полученных посылок заполнены верно
		p, ok := parcelMap[parcel.Number]
		assert.True(t, ok, "parcel not found by number %d", parcel.Number)
		assertParcelsEqual(t, p, parcel)
	}
}
