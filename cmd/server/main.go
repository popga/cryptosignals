package main

import (
	"cryptoapi/internal/api"
	"cryptoapi/internal/cache"
	"cryptoapi/internal/config"
	"cryptoapi/internal/logging"
	"log"
	"time"
)

/*
type KD struct {
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

type MKD struct {
	D []KD
}

func (k *MKD) UnmarshalJSON(data [] byte) error {
	var v [][]interface{}
	r := bytes.NewReader(data)
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	if err := decoder.Decode(&v); err != nil {
		fmt.Println(err)
	}
	k.D = make([]KD, len(v))
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
		quoteAssetVolume, err := strconv.ParseFloat(v[i][7].(string), 64)
		if err != nil {
			return err
		}
		quoteAssetVolumeBigFloat, _ := big.NewFloat(quoteAssetVolume / 10).SetMode(big.ToPositiveInf).SetPrec(2).Float64()
		numberOfTrades, err := v[i][8].(json.Number).Int64()
		if err != nil {
			return err
		}
		takerBuyBaseAssetVolume, err := strconv.ParseFloat(v[i][9].(string), 64)
		if err != nil {
			return err
		}
		takerBuyBaseAssetVolumeBigFloat, _ := big.NewFloat(takerBuyBaseAssetVolume / 10).SetMode(big.ToPositiveInf).SetPrec(2).Float64()
		takerBuyQuoteAssetVolume, err := strconv.ParseFloat(v[i][10].(string), 64)
		if err != nil {
			return err
		}
		takerBuyQuoteAssetVolumeBigFloat, _ := big.NewFloat(takerBuyQuoteAssetVolume / 10).SetMode(big.ToPositiveInf).SetPrec(2).Float64()

		k.D[i].OpenTime = openTime
		k.D[i].Open = open
		k.D[i].High = high
		k.D[i].Low = low
		k.D[i].Close = close
		k.D[i].Volume = volume
		k.D[i].CloseTime = closeTime
		k.D[i].QuoteAssetVolume = quoteAssetVolumeBigFloat
		k.D[i].NumberOfTrades = numberOfTrades
		k.D[i].TakerBuyBaseAssetVolume = takerBuyBaseAssetVolumeBigFloat
		k.D[i].TakerBuyQuoteAssetVolume = takerBuyQuoteAssetVolumeBigFloat
	}
	return nil
}

type getFunc func(*http.Client, string, string) (*MKD, error)

func getData(c *http.Client, ticker string, interval string) (*MKD, error) {
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s", ticker, interval)
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	data := new(MKD)
	if err := json.Unmarshal(raw, data); err != nil {
		return nil, err
	}
	return data, nil
}

func retryFunc(c *http.Client, ticker, interval string, fn getFunc) (*MKD, error) {
	count := 0

	t := time.NewTimer(time.Second)
	for {
		d, err := fn(c, ticker, interval)
		if err == nil {
			return d, nil
		}
		fmt.Printf("%s_%s failed: count: %d", ticker, interval, count)
		count++
		t.Reset(time.Second * time.Duration(count))

		if count >= 10 {
			return nil, errors.New("too many attempts")
		}
		<-t.C
	}
}

var (
	tickers   = []string{"BTCUSDT", "ETHUSDT", "XRPUSDT"}
	intervals = []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "12h", "1d", "1w", "1M"}
)

type Cache struct {
	C map[string]*MKD
	sync.Mutex
}

func New() *Cache {
	return &Cache{
		C:     make(map[string]*MKD),
		Mutex: sync.Mutex{},
	}
}

func (c *Cache) Set(k string, d *MKD) {
	c.Lock()
	c.C[k] = d
	c.Unlock()
}
func (c *Cache) Update(k string) {
	c.Lock()
	c.Unlock()
}

func (c *Cache) Exists(k string) bool {
	c.Lock()
	_, v := c.C[k]
	c.Unlock()
	if v {
		return true
	}
	return false
}

func start(cache *Cache) {
	c := &http.Client{
		Timeout: time.Second * 5,
	}
	//ticker := time.NewTicker(time.Millisecond * 5)
	for {
		for i := 0; i < len(tickers); i++ {
			for j := 0; j < len(intervals); j++ {
				t := fmt.Sprintf("%s_%s", tickers[i], intervals[j])
				fmt.Println(t)
				d, err := retryFunc(c, tickers[i], intervals[j], getData)
				if err != nil {
					fmt.Println(err)
					continue
				}
				cache.Set(t, d)
				//<-ticker.C
			}
		}
	}
}


*/
func main() {
	if err := config.Create(); err != nil {
		log.Fatal(err)
	}
	logFile, logger, err := logging.NewLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	cache := cache.New()
	cryptoapi := api.New(logger, cache, time.Millisecond*500)

	//cryptoapi.CollectOldData()
	//cryptoapi.LoadIntoCache()
	for {
		cryptoapi.CollectData()
	}
	//fmt.Println("test")
	//fmt.Println("wow")
	//data, err := api.ReadDataFromFile("BTCUSDT_2h_old_1578881978.gz")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//a, b, c := talib.Macd(data.Close, 12, 26, 9)

	//cryptoapi.TestRSI("BTCUSDT_2h", 10000, 10000, 1.00, 30, 70)
	//cryptoapi.TestEngulfing("BTCUSDT_5m", 1000, 1000, 1.00)

}
