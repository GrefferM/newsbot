import Foundation
import Testing

@testable import NewsbotCore

@Test func decodesAnAgentAnswerWithFindingsAndScenarios() throws {
    let json = """
        {"text":"Стан.","calls":["server_get","plan_scenario"],
         "findings":[{"id":"f1","asset":"BLUR","published":"2026-10-12","day":"2026-11-12","text":"Закриття сайту",
                      "source":"https://www.coindesk.com/x","outlet":"CoinDesk","others":["https://blur.io/blog/a-long-page-name-here","https://blur.io/blog/other"],
                      "unconfirmed":true,"approximate":true}],
         "scenarios":[{"condition":"Вийде делістинг","action":"шорт одразу","risk":"мінімальний","why":"рух відомий"}]}
        """

    let answer = try ServerJSON.decoder.decode(AgentAnswer.self, from: Data(json.utf8))

    let finding = answer.findings[0]

    #expect(
        finding.sources.map(finding.label(of:)) == ["CoinDesk", "blur.io/…ong-page-name-here", "blur.io/…other"])
    #expect(finding.stamp == "2026-10-12")
    #expect(answer.findings.first?.unconfirmed == true)
    #expect(answer.scenarios.first?.risk == .minimal)
}

@MainActor
@Test func eachAgentKeepsItsOwnThreadAndReportsFailures() async {
    let api = FakeAPI(answers: [
        .binance: AgentAnswer(text: "Падає UAI.", calls: ["binance_get"]),
        .osinter: AgentAnswer(text: "Висновок: чекати."),
    ])
    let analyst = Conversation(agent: .binance)
    let researcher = Conversation(agent: .osinter, topic: "BLUR")
    let risk = Conversation(agent: .risk)

    await analyst.send("Що падає?", with: api)
    await researcher.send("Чи шортити BLUR?", with: api)
    await risk.send("Ризики?", with: api)

    #expect(analyst.turns.map(\.question) == ["Що падає?"] && analyst.turns.map(\.answer) == ["Падає UAI."])
    #expect(analyst.turns.last?.calls == ["binance_get"] && researcher.turns.count == 1)
    #expect(risk.turns.last?.failed == true && !risk.working)
}

@MainActor
@Test func newsStaysWithTheCoinAndThePlanIsMade() async {
    let old = Finding(
        id: "p1", asset: "BLUR", day: "2026-06-18", text: "Monitoring Tag", source: "b", unconfirmed: false)
    let api = FakeAPI(
        answers: [
            .osinter: AgentAnswer(text: "Суть."),
            .risk: AgentAnswer(
                text: "Слабка монета.",
                scenarios: [Scenario(condition: "Делістинг", action: "шорт", risk: .minimal, why: "відомо")]
            ),
        ],
        pinned: [old]
    )
    let page = CoinPage(asset: "BLUR")

    await page.news.search("", with: api)

    #expect(page.news.news.map(\.id) == ["p1"] && page.news.loaded)
    #expect(page.news.researcher.turns.map(\.question) == [CoinNews.everything] && !page.news.searching)
    #expect(page.news.researcher.turns.map(\.answer) == ["Суть."])

    await page.plans.make("більше на новини", with: api)

    await page.news.forget(old, with: api)

    #expect(page.news.news.isEmpty)
    #expect(page.plans.plan?.scenarios.count == 1 && page.plans.plan?.summary == "Слабка монета.")
    #expect(page.plans.plan?.wish == "більше на новини" && !page.plans.planning)

    await page.trader.send("А якщо делістинг?", with: api)

    #expect(page.trader.turns.map(\.question) == ["А якщо делістинг?"] && page.trader.turns[0].failed)

    await page.trader.clear(with: api)

    #expect(page.trader.turns.isEmpty && page.plans.plan != nil)

    let mixed = EntryPlan(
        summary: "",
        scenarios: [
            Scenario(condition: "обережно", action: "чекати", risk: .minimal, why: ""),
            Scenario(condition: "сміливо", action: "вхід", risk: .high, why: ""),
            Scenario(condition: "помірно", action: "вхід", risk: .medium, why: ""),
        ],
        made: Date()
    )

    #expect(mixed.ranked.map(\.condition) == ["сміливо", "помірно", "обережно"])
}

@MainActor
@Test func theNavigatorRemembersTheCoinAndForgetsTheNewsOnceShown() throws {
    let defaults = try #require(UserDefaults(suiteName: "navigator-\(UUID())"))
    let navigator = Navigator(defaults: defaults)

    navigator.show(coin: "BANANA", news: "Запуск")
    navigator.shown(news: "Інше")

    #expect(navigator.coin == "BANANA" && navigator.news == "Запуск")

    navigator.shown(news: "Запуск")

    #expect(navigator.news == nil && Navigator(defaults: defaults).coin == "BANANA")
}

@MainActor
@Test func theChatComesBackFromTheServerAndKeepsTheLastTen() async {
    let past = (1...3).map {
        Exchange(at: Date(), question: "Питання \($0)", answer: "Відповідь \($0)", calls: ["market_screen"])
    }
    let api = FakeAPI(answers: [.binance: AgentAnswer(text: "Нова")], past: past)
    let analyst = Conversation(agent: .binance)

    await analyst.restore(from: api)

    #expect(analyst.turns.first?.question == "Питання 1" && analyst.turns.first?.answer == "Відповідь 1")
    #expect(analyst.turns.last?.calls == ["market_screen"])

    for _ in 1...9 {
        await analyst.send("Ще", with: api)
    }

    await analyst.restore(from: api)

    #expect(analyst.turns.count == 10 && analyst.turns.first?.question == "Питання 3")

    await analyst.send("Кандидати на шорт", with: api, reach: .scout)

    #expect(analyst.turns.last?.answer == "Воронка: Кандидати на шорт")

    await analyst.send("Швидкі ідеї", with: api, reach: .ideas)

    #expect(analyst.turns.last?.ideas.map(\.asset) == ["ZEC"] && analyst.turns.last?.ideas.first?.long == false)
    #expect(analyst.turns.last?.ideas.first?.facts == "ціна 1050.00 · фандинг −0.010% / 8 год")

    await analyst.clear(with: api)

    #expect(analyst.turns.isEmpty)
}

@MainActor
@Test func theTalkAboutACoinComesBackWithIt() async {
    let past = [
        Exchange(at: Date(), question: "Що з голосуванням?", answer: "Триває."),
        Exchange(at: Date(), question: "А якщо делістинг?", answer: "Тоді шорт."),
    ]
    let page = CoinPage(asset: "ZEC")

    await page.load(from: FakeAPI(past: past))

    let told = page.trader.turns

    #expect(told.map(\.question) == ["Що з голосуванням?", "А якщо делістинг?"] && told[1].answer == "Тоді шорт.")
    #expect(page.news.researcher.turns.count == 2)

    for index in 1...11 {
        await page.trader.send("Питання \(index)", with: FakeAPI(past: past))
    }

    #expect(page.trader.turns.count == 10 && page.trader.turns.last?.failed == true)
}

@MainActor
@Test func aPlanWithNumbersIsWatchedAndReleased() async {
    let api = FakeAPI()
    let plans = CoinPlans(asset: "ZEC")
    let loose = Scenario(condition: "якщо щось", action: "чекати", risk: .minimal, why: "")
    let sound = Scenario(
        condition: "закриття вище 1120", action: "лонг", risk: .medium, why: "", direction: "лонг", wakeAbove: 1120,
        sizePct: 7, stop: 1055, take: 1250)

    #expect(!loose.armable && loose.numbers == nil && sound.armable)
    #expect(sound.numbers == "лонг від 1120.00 · 7% рахунку · стоп 1055.00 · тейк 1250.00")

    let small = Scenario(
        condition: "c", action: "a", risk: .minimal, why: "w", direction: "лонг", wakeAbove: 1120, sizePct: 0.4,
        stop: 1055, take: 1250)

    #expect(small.numbers?.contains("· 0.4% рахунку ·") == true)

    let dated = Scenario(
        condition: "після голосування", action: "лонг", risk: .minimal, why: "", direction: "лонг", wakeAbove: 1075,
        sizePct: 5, stop: 1010, take: 1180, wakeAfter: "2026-11-16")

    let voted = Scenario(
        condition: "після голосування", action: "лонг", risk: .minimal, why: "", direction: "лонг", sizePct: 5,
        stop: 1010, take: 1180, wakeAfter: "2026-11-16")

    #expect(dated.numbers?.hasPrefix("лонг від 1075.00 з 16-11-2026 ·") == true)
    #expect(!voted.armable && voted.numbers == nil)

    await plans.arm(sound, with: api)

    #expect(plans.watch(of: sound)?.state == .armed && plans.watch(of: loose) == nil)

    if let watch = plans.watch(of: sound) {
        await plans.disarm(watch, with: api)
    }

    #expect(plans.watches.isEmpty)
}

@MainActor
@Test func aWatchedPlansNumbersAreEditedButNotItsCondition() async {
    let api = FakeAPI()
    let plans = CoinPlans(asset: "BANANA")
    let plan = Scenario(
        condition: "пробій 0.00936", action: "лонг", risk: .high, why: "", direction: "лонг", wakeAbove: 0.00936,
        sizePct: 10, stop: 0.0083, take: 0.013)

    await plans.arm(plan, with: api)

    guard let watch = plans.waiting.first else {
        Issue.record("not armed")
        return
    }

    var numbers = PlanNumbers(watch.plan)

    #expect(numbers.wake == "0.00936" && numbers.size == "10" && numbers.wakeTitle == "Ціна-поріг, від")

    numbers.stop = "0,0085"
    numbers.after = "2026-11-16"

    #expect(await plans.amend(watch, to: numbers, with: api))
    #expect(plans.waiting.first?.plan.stop == 0.0085 && plans.waiting.first?.plan.wakeAfter == "2026-11-16")
    #expect(plans.waiting.first?.plan.condition == "пробій 0.00936")

    numbers.take = "дорого"

    #expect(await plans.amend(watch, to: numbers, with: api) == false && plans.failure != nil)

    numbers.take = "0.013"
    numbers.wake = ""

    #expect(numbers.applied(to: watch.plan) == nil)
}

@MainActor
@Test func aCoinPageOutlivesItsWindowWithATalkStillWaiting() async {
    let pages = CoinPages()
    let first = pages.page(for: "HUMA")
    let api = FakeAPI()

    await first.trader.send("Проаналізуй токен", with: api)

    #expect(pages.page(for: "HUMA") === first)
    #expect(pages.page(for: "HUMA").trader.turns.count == 1)
    #expect(pages.page(for: "ZEC") !== first)
}

@Test func aPlanShowsTheOwnersWishOrTheBareRequest() {
    #expect(CoinPlans.shown("лише лонги, малий розмір") == "лише лонги, малий розмір")
    #expect(CoinPlans.shown(nil) == "Склади плани входу")
    #expect(CoinPlans.shown("") == "Склади плани входу")
}
