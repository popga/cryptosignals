package api

import (
	"bytes"
	"compress/gzip"
	"cryptoapi/internal/cache"
	"cryptoapi/internal/helpers"
	"cryptoapi/internal/logging"
	"cryptoapi/internal/talib"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	Tickers   = []string{"BTCUSDT", "ETHUSDT", "XRPUSDT"}
	Intervals = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d", "1w", "1M"}
)

//type CryptoGetDataFromBinance func(string, string) (*BinanceData, error)
type CryptoGetDataFromBinance func(string, string, int64) ([]byte, error)

type BinanceKlineData struct {
	OpenTime                 int64   `json:"open_time"`
	Open                     float64 `json:"open"`
	High                     float64 `json:"high"`
	Low                      float64 `json:"low"`
	Close                    float64 `json:"close"`
	Volume                   float64 `json:"volume"`
	CloseTime                int64   `json:"close_time"`
	QuoteAssetVolume         float64 `json:"quote_asset_volume"`
	NumberOfTrades           int64   `json:"number_of_trades"`
	TakerBuyBaseAssetVolume  float64 `json:"taker_buy_base_asset_volume"`
	TakerBuyQuoteAssetVolume float64 `json:"taker_buy_quote_asset_volume"`
}

type BinanceData struct {
	Data []BinanceKlineData
}

type klineData struct {
	OpenTime  []int64
	Open      []float64
	High      []float64
	Low       []float64
	Close     []float64
	Volume    []float64
	CloseTime []int64
}

//func (binanceData *BinanceData) UnmarshalJSON(data [] byte) error {
func (d *klineData) UnmarshalJSON(data [] byte) error {
	var v [][]interface{}
	r := bytes.NewReader(data)
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	if err := decoder.Decode(&v); err != nil {
		return err
	}
	if len(v) < 1 {
		return errors.New("empty data")
	}
	//binanceData.Data = make([]BinanceKlineData, len(v))
	d.OpenTime = make([]int64, len(v))
	d.Open = make([]float64, len(v))
	d.High = make([]float64, len(v))
	d.Low = make([]float64, len(v))
	d.Close = make([]float64, len(v))
	d.Volume = make([]float64, len(v))
	d.CloseTime = make([]int64, len(v))
	for i := 0; i < len(v); i++ {
		openTime, err := v[i][0].(json.Number).Int64()
		if err != nil {
			return err
		}
		open, err := strconv.ParseFloat(v[i][1].(string), 64)
		if err != nil {
			return err
		}
		high, err := strconv.ParseFloat(v[i][2].(string), 64)
		if err != nil {
			return err
		}
		low, err := strconv.ParseFloat(v[i][3].(string), 64)
		if err != nil {
			return err
		}
		close, err := strconv.ParseFloat(v[i][4].(string), 64)
		if err != nil {
			return err
		}
		volume, err := strconv.ParseFloat(v[i][5].(string), 64)
		if err != nil {
			return err
		}
		closeTime, err := v[i][6].(json.Number).Int64()
		if err != nil {
			return err
		}
		d.OpenTime[i] = openTime
		d.Open[i] = open
		d.High[i] = high
		d.Low[i] = low
		d.Close[i] = close
		d.Volume[i] = volume
		d.CloseTime[i] = closeTime
	}
	return nil
}

type CryptoAPI struct {
	HttpClient *http.Client
	*logging.Logger
	Cache *cache.Cache
	Delay time.Duration
}

func New(logger *logging.Logger, cache *cache.Cache, delay time.Duration) *CryptoAPI {
	return &CryptoAPI{
		HttpClient: &http.Client{},
		Logger:     logger,
		Cache:      cache,
		Delay:      delay,
	}
}

func (cryptoapi *CryptoAPI) FormatTickerKey(ticker, interval string) string {
	return fmt.Sprintf("%s_%s", ticker, interval)
}

func (cryptoapi *CryptoAPI) CollectDataFromBinance(ticker, interval string, endTime int64) ([]byte, error) {
	if ticker == "" || interval == "" {
		return nil, errors.New("parameters not provided")
	}
	var url string
	if endTime == 0 {
		url = fmt.Sprintf(viper.GetString("binance-without-endtime"), ticker, interval)
	} else {
		url = fmt.Sprintf(viper.GetString("binance-with-endtime"), ticker, interval, endTime)
	}
	resp, err := cryptoapi.HttpClient.Get(url)
	if err != nil {
		return nil, err
	}
	b := new(bytes.Buffer)
	if _, err := io.Copy(b, resp.Body); err != nil {
		return nil, err
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func processData(raw []byte) (*klineData, error) {
	data := new(klineData)
	if err := json.Unmarshal(raw, data); err != nil {
		return nil, err
	}
	return data, nil
}

//func RetryFunc(ticker, interval string, fn CryptoGetDataFromBinance) (*BinanceData, error) {
func RetryFunc(ticker, interval string, endTime int64, fn CryptoGetDataFromBinance) (*klineData, error) {
	count := 0
	t := time.NewTimer(time.Second)
	for {
		d, err := fn(ticker, interval, endTime)
		if err == nil {
			d2, err := processData(d)
			if err == nil {
				return d2, nil
			}
		}
		fmt.Printf("%s_%s failed: count: %d, error: %s", ticker, interval, count, err.Error())
		count++
		t.Reset(time.Second * time.Duration(count))
		if count >= 5 {
			return nil, errors.New("too many attempts")
		}
		<-t.C
	}
}

func createFileAndWrite(name string, data *klineData) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := helpers.CreateDirIfNotExist(viper.GetString("base.data.folder")); err != nil {
		return err
	}
	fullPath := filepath.Join(cwd, viper.GetString("base.data.folder"), fmt.Sprintf("%s.gz", name))
	f, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	b := new(bytes.Buffer)
	if err := gob.NewEncoder(b).Encode(data); err != nil {
		return err
	}
	gzipWriter, err := gzip.NewWriterLevel(f, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := gzipWriter.Write(b.Bytes()); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	fmt.Println("written", f.Name())
	return nil
}
func ReadDataFromFile(name string) (*klineData, error) {
	f, err := os.OpenFile(name, os.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}
	gzipReader, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	if err := gzipReader.Close(); err != nil {
		return nil, err
	}
	data := new(klineData)
	if err := gob.NewDecoder(gzipReader).Decode(data); err != nil {
		return nil, err
	}
	return data, nil
}

func findFile(pattern string) (string, error) {
	fullPath := fmt.Sprintf("%s/%s", viper.GetString("base.data.folder"), pattern)
	m, err := filepath.Glob(fullPath)
	if err != nil {
		return "", err
	}
	if m[0] == "" {
		return "", errors.New("could not find file")
	}
	return m[0], nil
}

func (cryptoapi *CryptoAPI) LoadIntoCache() {
	for i := 0; i < len(Tickers); i++ {
		for j := 0; j < len(Intervals); j++ {
			f, err := findFile(fmt.Sprintf("%s_%s_*.gz", Tickers[i], Intervals[j]))
			if err != nil {
				cryptoapi.WithError(err).Debug("findFile error, will continue")
				continue
			}
			t1, err := ReadDataFromFile(f)
			if err != nil {
				cryptoapi.WithError(err).Debug("failed reading data from file, will continue")
				continue
			}
			fn := fmt.Sprintf("%s_%s_old", Tickers[i], Intervals[j])
			cryptoapi.Cache.Set(fn, t1)
		}
	}
}

// StartOldData collects old data from Binance
func (cryptoapi *CryptoAPI) CollectOldData() {
	if err := helpers.DeleteDir(); err != nil {
		fmt.Println(err)
		return
	}
	d := new(klineData)
	var lastOpenTime int64 = 0
	for i := 0; i < len(Tickers); i++ {
		for j := 0; j < len(Intervals); j++ {
			cryptoapi.Debugf("processing %s_%s", Tickers[i], Intervals[j])
			lastOpenTime = 0
			d = new(klineData)
			for {
				data, err := RetryFunc(Tickers[i], Intervals[j], lastOpenTime, cryptoapi.CollectDataFromBinance)
				if err != nil {
					fmt.Println("MUIE", err)
					cryptoapi.WithError(err).Debugf("%s_%s failed too many times", Tickers[i], Intervals[i])
					continue
				}
				if lastOpenTime == data.OpenTime[0] || len(d.OpenTime) >= 50000 {
					fmt.Println("WRITING", Tickers[i], Intervals[j])
					if err := createFileAndWrite(fmt.Sprintf("%s_%s_old_%d", Tickers[i], Intervals[j], time.Now().Unix()), d); err != nil {
						fmt.Println("PIZDA", err)
						d = new(klineData)
						cryptoapi.WithError(err).Debugf("failed creating file: %s_%s_%d", Tickers[i], Intervals[j], time.Now().Unix())
						//continue
						break
					}
					d = new(klineData)
					break
				}
				lastOpenTime = data.OpenTime[0]
				d.OpenTime = append(data.OpenTime[:len(data.OpenTime)-1], d.OpenTime...)
				d.Open = append(data.Open[:len(data.Open)-1], d.Open...)
				d.High = append(data.High[:len(data.High)-1], d.High...)
				d.Low = append(data.Low[:len(data.Low)-1], d.Low...)
				d.Close = append(data.Close[:len(data.Close)-1], d.Close...)
				d.Volume = append(data.Volume[:len(data.Volume)-1], d.Volume...)
				d.CloseTime = append(data.CloseTime[:len(data.CloseTime)-1], d.CloseTime...)
			}
		}
	}
}
func rev(s []float64) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Start collects recent data from binance
func (cryptoapi *CryptoAPI) CollectData() {
	timer := time.NewTicker(cryptoapi.Delay)
	for i := 0; i < len(Tickers); i++ {
		for j := 0; j < len(Intervals); j++ {
			cryptoapi.Debugf("processing %s_%s", Tickers[i], Intervals[j])
			data, err := RetryFunc(Tickers[i], Intervals[j], 0, cryptoapi.CollectDataFromBinance)
			if err != nil {
				cryptoapi.WithError(err).Debugf("%s_%s failed too many times, will skip", Tickers[i], Intervals[i])
				continue
			}
			key := cryptoapi.FormatTickerKey(Tickers[i], Intervals[j])
			cryptoapi.CalculateIndicators(data)
			cryptoapi.Cache.Set(key, data)
			<-timer.C
		}
	}
}

func (cryptoapi *CryptoAPI) StartWithLoop() {
	for {
		cryptoapi.CollectData()
	}
}

func (cryptoapi *CryptoAPI) CalculateIndicators(data *klineData) {
	rsi14 := talib.Rsi(data.Close, 14)
	if rsi14[len(rsi14)-1] < 30 {
		// send to subscribers
	}
	if rsi14[len(rsi14)-1] > 70 {
		// send to subs
	}
}

func (cryptoapi *CryptoAPI) TestRSI(ticker string, balance float64, tradeSize int64, stopLossPercent float64, buySignal, sellSignal float64) {
	t := cryptoapi.Cache.Get(ticker)
	tickerData := t.(*klineData)

	var currentBalance = balance
	var boughtAt float64
	var soldAt float64
	var orderPlaced bool
	profitPercent := 0.00
	maxDif := 5.0

	rsi14 := talib.Rsi(tickerData.Close, 14)
	for i := 8; i < len(tickerData.Close); i++ {
		if stopLossPercent != 0.00 && orderPlaced == true {
			//onePercent := boughtAt * 0.01
			onePercent := boughtAt * (stopLossPercent / 100)
			stopLossPrice := boughtAt - onePercent
			if tickerData.Close[i] <= stopLossPrice {
				orderPlaced = false
				soldAt = tickerData.Close[i]
				currentBalance = currentBalance - (float64(tradeSize) * (stopLossPercent / 100))
				cryptoapi.Debugf("date: %s, stoploss hit at: %.2f, trade size: $%.2f, current balance: $%.2f", time.Unix(tickerData.CloseTime[i]/1000, 0), stopLossPrice, float64(tradeSize), currentBalance)
			}
		}
		if rsi14[i] <= buySignal+maxDif && rsi14[i] >= buySignal-maxDif {
			if orderPlaced {
				cryptoapi.Debug("an order is already open")
				continue
			}
			orderPlaced = true
			boughtAt = tickerData.Close[i]
			cryptoapi.Debugf("date: %s, bought at: %.2f, rsi (14): %.2f, trade size: $%.2f, current balance: $%.2f", time.Unix(tickerData.CloseTime[i]/1000, 0), boughtAt, rsi14[i], float64(tradeSize), currentBalance)
		} else if rsi14[i] <= sellSignal+maxDif && rsi14[i] >= sellSignal-maxDif {
			if orderPlaced {
				orderPlaced = false
				soldAt = tickerData.Close[i]
				profitPercent = ((soldAt - boughtAt) / boughtAt) * 100
				if profitPercent > 0.00 {
					currentBalance = currentBalance + ((float64(tradeSize) * profitPercent) / 100)
				} else if profitPercent < 0.00 {
					currentBalance = currentBalance - ((float64(tradeSize) * math.Abs(profitPercent)) / 100)
				}
				cryptoapi.Debugf("date: %s, sold at: %.2f, rsi (14): %.2f, trade size: $%.2f, profit: $%0.2f, current balance: $%.2f", time.Unix(tickerData.CloseTime[i]/1000, 0),
					soldAt, rsi14[i], float64(tradeSize), ((profitPercent * float64(tradeSize)) / 100), currentBalance)
			}
		}
	}
}
func (cryptoapi *CryptoAPI) TestEngulfing(ticker string, balance float64, tradeSize int64, stopLossPercent float64) {
	t := cryptoapi.Cache.Get(ticker)
	tickerData := t.(*klineData)

	var currentBalance = balance
	var boughtAt float64
	var soldAt float64
	var orderPlaced bool
	profitPercent := 0.00

	engulfing := talib.CdlEngulfing(tickerData.Open, tickerData.High, tickerData.Low, tickerData.Close)
	for i := 8; i < len(tickerData.Close); i++ {
		if stopLossPercent != 0.00 && orderPlaced == true {
			//onePercent := boughtAt * 0.01
			onePercent := boughtAt * (stopLossPercent / 100)
			stopLossPrice := boughtAt - onePercent
			if tickerData.Close[i] <= stopLossPrice {
				orderPlaced = false
				soldAt = tickerData.Close[i]
				currentBalance = currentBalance - (float64(tradeSize) * (stopLossPercent / 100))
				cryptoapi.Debugf("date: %s, stoploss hit at: %.2f, trade size: $%.2f, current balance: $%.2f", time.Unix(tickerData.CloseTime[i]/1000, 0), stopLossPrice, float64(tradeSize), currentBalance)
			}
		}
		if engulfing[i] > 0 {
			if orderPlaced {
				cryptoapi.Debug("an order is already open")
				continue
			}
			orderPlaced = true
			boughtAt = tickerData.Close[i]
			cryptoapi.Debugf("date: %s, bought at: %.2f, engulfing: %d, trade size: $%.2f, current balance: $%.2f", time.Unix(tickerData.CloseTime[i]/1000, 0), boughtAt, engulfing[i], float64(tradeSize), currentBalance)
		} else if engulfing[i] < 0 {
			if orderPlaced {
				orderPlaced = false
				soldAt = tickerData.Close[i]
				profitPercent = ((soldAt - boughtAt) / boughtAt) * 100
				if profitPercent > 0.00 {
					currentBalance = currentBalance + ((float64(tradeSize) * profitPercent) / 100)
				} else if profitPercent < 0.00 {
					currentBalance = currentBalance - ((float64(tradeSize) * math.Abs(profitPercent)) / 100)
				}
				cryptoapi.Debugf("date: %s, sold at: %.2f, engulfing: %d, trade size: $%.2f, profit: $%0.2f, current balance: $%.2f", time.Unix(tickerData.CloseTime[i]/1000, 0),
					soldAt, engulfing[i], float64(tradeSize), ((profitPercent * float64(tradeSize)) / 100), currentBalance)
			}
		}
	}
}
