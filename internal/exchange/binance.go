package exchange

import (
	"regexp"

	"newsbot/internal/news"
)

const (
	KindVoteDelisting = "Повний делістинг (Vote to Delist)"
	KindFullDelisting = "Повний делістинг Binance"
	KindPerpDelisting = "Делістинг перпа Binance"
	KindMonitoringTag = "Monitoring Tag Binance"
	KindVoteStarted   = "Vote to Delist Binance"
	KindEarnAdd       = "Binance додає в Earn і Margin"
)

var (
	binanceFullDelist = regexp.MustCompile(`(?i)^Binance Will Delist (.+?) on \d{4}-\d{2}-\d{2}`)
	binanceVoteDelist = regexp.MustCompile(`(?i)Vote to Delist.*Will Delist (.+?) on \d{4}-\d{2}-\d{2}`)
	binanceNotSpot    = regexp.MustCompile(`(?i)margin|options|leveraged|alpha|earn|loan|collateral`)
	binancePerpDelist = regexp.MustCompile(`(?i)^Binance Futures Will Delist`)
	binanceMonitoring = regexp.MustCompile(`(?i)Monitoring Tag to (?:Include )?(.+?)(?:,? and Remove|, Remove| on \d{4}-|$)`)
	binanceVoteStart  = regexp.MustCompile(`(?i)Vote to Delist`)
	binanceEarnAdd    = regexp.MustCompile(`(?i)^Binance Will Add (.+?) on Earn`)
)

func ClassifyBinance(title string) (Classification, bool) {
	switch {
	case binanceVoteDelist.MatchString(title):
		return shortTrigger(KindVoteDelisting, news.Critical, listedIn(binanceVoteDelist, title))
	case binanceFullDelist.MatchString(title) && !binanceNotSpot.MatchString(title):
		return shortTrigger(KindFullDelisting, news.Critical, listedIn(binanceFullDelist, title))
	case binancePerpDelist.MatchString(title):
		return shortTrigger(KindPerpDelisting, news.High, news.SymbolsFromUSDTPairs(title))
	case binanceMonitoring.MatchString(title):
		return shortTrigger(KindMonitoringTag, news.Critical, listedIn(binanceMonitoring, title))
	case binanceVoteStart.MatchString(title):
		return shortTrigger(KindVoteStarted, news.Normal, nil)
	case binanceEarnAdd.MatchString(title):
		return shortTrigger(KindEarnAdd, news.High, listedIn(binanceEarnAdd, title))
	}

	return ignored()
}

func listedIn(re *regexp.Regexp, title string) []news.Symbol {
	return news.SymbolsFromList(re.FindStringSubmatch(title)[1])
}
