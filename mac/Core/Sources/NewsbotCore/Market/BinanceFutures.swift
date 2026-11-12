import Foundation

public struct BinanceFutures: MarketFeed {
    private static let rest = "https://fapi.binance.com"
    private static let testnetRest = "https://testnet.binancefuture.com"
    private static let stream = "wss://fstream.binance.com/market/ws/"

    private let session: URLSession

    public init(session: URLSession = .shared) {
        self.session = session
    }

    public func candles(
        contract: String, interval: ChartInterval, limit: Int, before: Date?
    ) async throws -> [Candle] {
        var query = ["symbol": contract, "interval": interval.rawValue, "limit": String(limit)]

        if let before {
            query["endTime"] = Self.millis(before.addingTimeInterval(-1))
        }

        return try Self.parseCandles(try await get(Self.rest + "/fapi/v1/klines", query))
    }

    public func updates(contract: String, interval: ChartInterval) -> AsyncThrowingStream<Candle, Error> {
        let url = URL(string: Self.stream + contract.lowercased() + "@kline_" + interval.rawValue)!
        let task = session.webSocketTask(with: url)

        return AsyncThrowingStream { continuation in
            let reader = Task {
                do {
                    while !Task.isCancelled {
                        if case .string(let text) = try await task.receive(), let candle = Self.parseUpdate(text) {
                            continuation.yield(candle)
                        }
                    }
                } catch {
                    continuation.finish(throwing: error)
                }
            }

            continuation.onTermination = { _ in
                reader.cancel()
                task.cancel(with: .goingAway, reason: nil)
            }

            task.resume()
        }
    }

    public func day(contract: String) async throws -> DayStats {
        let data = try await get(Self.rest + "/fapi/v1/ticker/24hr", ["symbol": contract])
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any]

        guard
            let change = Self.number(object?["priceChangePercent"]),
            let volume = Self.number(object?["quoteVolume"])
        else {
            throw ClientError.status(0)
        }

        return DayStats(change: change / 100, volume: volume)
    }

    public func price(contract: String, testnet: Bool) async throws -> Double {
        let host = testnet ? Self.testnetRest : Self.rest
        let data = try await get(host + "/fapi/v1/ticker/price", ["symbol": contract])
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any]

        guard let price = Self.number(object?["price"]) else {
            throw ClientError.status(0)
        }

        return price
    }

    public func book(contract: String) async throws -> OrderBook {
        let data = try await get(Self.rest + "/fapi/v1/depth", ["symbol": contract, "limit": "500"])

        return try Self.parseBook(data)
    }

    public func derivatives(
        contract: String, interval: ChartInterval, span: ClosedRange<Date>?
    ) async throws -> Derivatives {
        let period = interval.derivativesStep
        let kept = Date().addingTimeInterval(-Derivatives.history)
        let until = span?.upperBound ?? Date()
        let since = span?.lowerBound ?? until.addingTimeInterval(-period.seconds * Double(Derivatives.points))
        var history = ["symbol": contract, "period": period.name, "limit": String(Derivatives.points)]
        var paid = ["symbol": contract, "startTime": Self.millis(since), "limit": "1000"]

        if span != nil {
            history["startTime"] = Self.millis(max(since, kept))
            history["endTime"] = Self.millis(until)
            paid["endTime"] = Self.millis(until)
        }

        async let interest = recent(Self.rest + "/futures/data/openInterestHist", history, until: until, kept: kept)
        async let funding = get(Self.rest + "/fapi/v1/fundingRate", paid)
        async let ratio = recent(
            Self.rest + "/futures/data/globalLongShortAccountRatio", history, until: until, kept: kept)

        return Derivatives(
            openInterest: Self.parseSeries(try await interest, time: "timestamp", value: "sumOpenInterestValue"),
            funding: Self.parseSeries(try await funding, time: "fundingTime", value: "fundingRate"),
            longs: Self.parseSeries((try? await ratio) ?? Data(), time: "timestamp", value: "longAccount")
        )
    }

    private func recent(_ address: String, _ query: [String: String], until: Date, kept: Date) async throws -> Data {
        guard until > kept else {
            return Data()
        }

        return try await get(address, query)
    }

    public static func parseBook(_ data: Data) throws -> OrderBook {
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any]

        return OrderBook(bids: levels(object?["bids"]), asks: levels(object?["asks"]))
    }

    public static func parseSeries(_ data: Data, time: String, value: String) -> [SeriesPoint] {
        let rows = (try? JSONSerialization.jsonObject(with: data)) as? [[String: Any]] ?? []

        return rows.compactMap { row in
            guard let moment = row[time] as? Double, let number = number(row[value]) else {
                return nil
            }

            return SeriesPoint(time: Date(timeIntervalSince1970: moment / 1000), value: number)
        }
    }

    private static func levels(_ raw: Any?) -> [BookLevel] {
        (raw as? [[Any]] ?? []).compactMap { row in
            guard row.count > 1, let price = number(row[0]), let quantity = number(row[1]) else {
                return nil
            }

            return BookLevel(price: price, quantity: quantity)
        }
    }

    public static func parseCandles(_ data: Data) throws -> [Candle] {
        let rows = try JSONSerialization.jsonObject(with: data) as? [[Any]] ?? []

        return rows.compactMap { row in
            guard row.count > 7, let opened = row[0] as? Double else {
                return nil
            }

            return candle(time: opened, row[1], row[2], row[3], row[4], row[7])
        }
    }

    public static func parseUpdate(_ text: String) -> Candle? {
        guard
            let object = try? JSONSerialization.jsonObject(with: Data(text.utf8)) as? [String: Any],
            let k = object["k"] as? [String: Any],
            let opened = k["t"] as? Double
        else {
            return nil
        }

        return candle(time: opened, k["o"], k["h"], k["l"], k["c"], k["q"])
    }

    private static func candle(time: Double, _ o: Any?, _ h: Any?, _ l: Any?, _ c: Any?, _ v: Any?) -> Candle? {
        guard let open = number(o), let high = number(h), let low = number(l), let close = number(c),
            let volume = number(v)
        else {
            return nil
        }

        return Candle(
            time: Date(timeIntervalSince1970: time / 1000),
            open: open,
            high: high,
            low: low,
            close: close,
            volume: volume
        )
    }

    private static func millis(_ moment: Date) -> String {
        String(Int(moment.timeIntervalSince1970 * 1000))
    }

    private static func number(_ value: Any?) -> Double? {
        (value as? String).flatMap(Double.init)
    }

    private func get(_ address: String, _ query: [String: String]) async throws -> Data {
        var url = URL(string: address)!
        url.append(queryItems: query.sorted { $0.key < $1.key }.map { URLQueryItem(name: $0.key, value: $0.value) })

        if let host = url.host(), Brake.shared.holds(host) {
            throw ClientError.held
        }

        let (data, response) = try await session.data(from: url)

        if let http = response as? HTTPURLResponse, http.statusCode != 200 {
            if http.statusCode == 429 || http.statusCode == 418, let host = url.host() {
                Brake.shared.hold(host, asked: http.value(forHTTPHeaderField: "Retry-After").flatMap(Double.init))
            }

            throw ClientError.status(http.statusCode)
        }

        return data
    }
}
