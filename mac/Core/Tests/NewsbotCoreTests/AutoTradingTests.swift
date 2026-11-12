import Foundation
import Testing

@testable import NewsbotCore

@Test func aStrategyShowsTheRecordTheServerCounted() throws {
    let json = """
        {"strategies": [{"id": "sample", "name": "Зразок", "about": "Зразок стратегії.",
          "terms": [{"name": "Стоп", "value": "20%"}], "on": true,
          "record": {"open": 1, "closed": 2, "won": 1, "lost": 1, "pnl_usd": 0.5},
          "rule": {"Take": 0.06, "Hold": 3600000000000, "Leverage": 5}}]}
        """
    let auto = try JSONDecoder().decode(AutoTrading.self, from: Data(json.utf8))
    let strategy = try #require(auto.strategies.first)

    #expect(strategy.terms == [StrategyTerm(name: "Стоп", value: "20%")])
    #expect(strategy.rule == ["Take": 0.06, "Hold": 3_600_000_000_000, "Leverage": 5])
    #expect(auto.countLine == "увімкнено 1 з 1")
    #expect(strategy.record.line == "Закрито 2: у плюс 1 (50%), разом $0.50 · відкрито 1")
    #expect(StrategyRecord(open: 2).line == "Відкрито угод: 2, закритих ще немає")
}

@Test func aStrategyWithoutTermsStillReads() throws {
    let json = """
        {"strategies": [{"id": "sample", "name": "Зразок", "about": "Зразок стратегії.", "on": false}]}
        """
    let auto = try JSONDecoder().decode(AutoTrading.self, from: Data(json.utf8))

    #expect(auto.strategies.first?.terms == [])
    #expect(auto.strategies.first?.rule == [:])
    #expect(auto.strategies.first?.record.line == "Угод ще не було")
    #expect(auto.running == false)
}
