import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { FormEvent } from 'react'
import { queryKeys } from '../api/queryKeys'
import { OperationStatisticsPanel } from '../components/analytics/OperationStatisticsPanel'
import { RequestStatus } from '../components/RequestStatus'
import type { OperationStatisticsResponse } from '../types/analytics'
import type { CardPaymentResponse, CardResponse, CardTransferResponse, CloseCardResponse } from '../types/card'
import { emptyState, type RequestState } from '../types/common'
import {
  formatCardNumber,
  isCardClosed,
} from '../utils/format'
import { Button } from '../components/ui/Button'
import { CardListPanel } from '../features/cards/CardListPanel'
import { Card } from '../components/ui/Card'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { OperationConfirmDialog } from '../components/ui/OperationConfirmDialog'
import { useCards } from '../hooks/useCards'
import { useMfaFlow } from '../hooks/useMfaFlow'
import { useToast } from '../hooks/useToast'
import { validateAmount } from '../utils/validation'

type CardsPageProps = {
  token: string
  sharedAccountId: string
}

const formatCardForDisplay = (card: CardResponse): CardResponse => ({
  ...card,
  number: card.number ? formatCardNumber(card.number) : card.number,
  masked_number: formatCardNumber(card.masked_number),
})

export function CardsPage({ token, sharedAccountId }: CardsPageProps) {
  const { showToast } = useToast()
  const queryClient = useQueryClient()
  const cardsDomain = useCards(token)
  const cardPaymentMfaFlow = useMfaFlow(token)
  const cardTransferMfaFlow = useMfaFlow(token)
  const cardRevealMfaFlow = useMfaFlow(token)
  const [cardsState, setCardsState] = useState<RequestState>(emptyState)
  const [createCardState, setCreateCardState] = useState<RequestState>(emptyState)
  const [cardRevealMfaState, setCardRevealMfaState] = useState<RequestState>(emptyState)
  const [cardRevealState, setCardRevealState] = useState<RequestState>(emptyState)
  const [cardPaymentMfaState, setCardPaymentMfaState] = useState<RequestState>(emptyState)
  const [cardPaymentState, setCardPaymentState] = useState<RequestState>(emptyState)
  const [cardTransferMfaState, setCardTransferMfaState] = useState<RequestState>(emptyState)
  const [cardTransferState, setCardTransferState] = useState<RequestState>(emptyState)
  const [cardStatisticsState, setCardStatisticsState] = useState<RequestState>(emptyState)
  const [cardCloseState, setCardCloseState] = useState<RequestState>(emptyState)

  const [cards, setCards] = useState<CardResponse[]>([])
  const [selectedCardId, setSelectedCardId] = useState('')
  const [selectedCard, setSelectedCard] = useState<CardResponse | null>(null)
  const [createCardAccountId, setCreateCardAccountId] = useState(sharedAccountId)
  const [createdCard, setCreatedCard] = useState<CardResponse | null>(null)

  const [cardRevealMfaCode, setCardRevealMfaCode] = useState('')
  const [revealedCardDetails, setRevealedCardDetails] = useState<CardResponse | null>(null)

  const [cardPaymentAmount, setCardPaymentAmount] = useState('100.00')
  const [cardPaymentCVV, setCardPaymentCVV] = useState('')
  const [cardPaymentMfaCode, setCardPaymentMfaCode] = useState('')
  const [cardPaymentDescription, setCardPaymentDescription] = useState('Card payment')

  const [cardTransferToCardNumber, setCardTransferToCardNumber] = useState('')
  const [cardTransferAmount, setCardTransferAmount] = useState('100.00')
  const [cardTransferCVV, setCardTransferCVV] = useState('')
  const [cardTransferMfaCode, setCardTransferMfaCode] = useState('')
  const [cardTransferDescription, setCardTransferDescription] = useState('Card transfer')
  const [cardStatisticsLimit, setCardStatisticsLimit] = useState('100')

  const [cardPaymentResult, setCardPaymentResult] = useState<CardPaymentResponse | null>(null)
  const [cardTransferResult, setCardTransferResult] = useState<CardTransferResponse | null>(null)
  const [cardOperationStatistics, setCardOperationStatistics] = useState<OperationStatisticsResponse | null>(null)
  const [cardCloseResult, setCardCloseResult] = useState<CloseCardResponse | null>(null)
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false)
  const [cardTransferConfirmOpen, setCardTransferConfirmOpen] = useState(false)

  useEffect(() => {
    if (sharedAccountId && !createCardAccountId) {
      setCreateCardAccountId(sharedAccountId)
    }
  }, [sharedAccountId, createCardAccountId])

  useEffect(() => {
    const cachedCards = cardsDomain.listQuery.data
    if (!cachedCards) {
      return
    }

    setCards(cachedCards)

    if (cachedCards.length === 0) {
      setSelectedCardId('')
      setSelectedCard(null)
      return
    }

    const cardToSelect =
      cachedCards.find((item) => String(item.id) === selectedCardId) ||
      (selectedCard
        ? cachedCards.find((item) => item.id === selectedCard.id)
        : undefined) ||
      cachedCards[0]

    setSelectedCardId(String(cardToSelect.id))
    setSelectedCard(cardToSelect)
  }, [cardsDomain.listQuery.data])

  const requireToken = (setState: (state: RequestState) => void): boolean => {
    if (token) {
      return true
    }

    setState({ loading: false, error: 'Сначала нужно войти в систему.', success: '' })
    return false
  }

  const selectedCardIDNumber = (): number | null => {
    const id = Number(selectedCardId)
    return Number.isInteger(id) && id > 0 ? id : null
  }

  const selectCard = (card: CardResponse) => {
    setSelectedCardId(String(card.id))
    setSelectedCard(card)
    setCardPaymentResult(null)
    setCardTransferResult(null)

    const statisticsLimit = Number(cardStatisticsLimit)
    const cachedStatistics = Number.isInteger(statisticsLimit)
      ? queryClient.getQueryData<OperationStatisticsResponse>(
        queryKeys.cards.operationStatistics(card.id, statisticsLimit),
      )
      : undefined
    setCardOperationStatistics(cachedStatistics || null)

    setCardStatisticsState(emptyState)
    setCardCloseResult(null)
    setCardRevealMfaCode('')
    setRevealedCardDetails(null)
    setCardRevealState(emptyState)
    setCardRevealMfaState(emptyState)
  }

  const upsertCard = (card: CardResponse) => {
    setCards((current) => {
      const exists = current.some((item) => item.id === card.id)
      return exists
        ? current.map((item) => (item.id === card.id ? { ...item, ...card } : item))
        : [card, ...current]
    })
    selectCard(card)
  }

  const updateCardListWithoutRevealedNumber = (card: CardResponse) => {
    const cardForList: CardResponse = { ...card, number: undefined }

    setCards((current) => {
      const exists = current.some((item) => item.id === card.id)
      return exists
        ? current.map((item) => (item.id === card.id ? { ...item, ...cardForList } : item))
        : [cardForList, ...current]
    })
  }

  const applyClosedCard = (response: CloseCardResponse) => {
    setCards((current) =>
      current.map((card) =>
        card.id === response.id
          ? { ...card, status: response.status, closed_at: response.closed_at }
          : card,
      ),
    )

    setSelectedCard((card) =>
      card && card.id === response.id
        ? { ...card, status: response.status, closed_at: response.closed_at }
        : card,
    )
  }

  const loadCards = async () => {
    if (!requireToken(setCardsState)) {
      return
    }

    setCardsState({ loading: true, error: '', success: '' })

    try {
      const result = await cardsDomain.listQuery.refetch()
      if (result.error) {
        throw result.error
      }
      const list = Array.isArray(result.data) ? result.data : []
      setCards(list)
      setCardsState({ loading: false, error: '', success: 'Список карт загружен.' })

      if (list.length > 0) {
        selectCard(list.find((item) => String(item.id) === selectedCardId) || list[0])
      } else {
        setSelectedCardId('')
        setSelectedCard(null)
      }
    } catch (error) {
      setCardsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load cards',
        success: '',
      })
    }
  }

  const createCard = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setCreateCardState)) {
      return
    }

    const accountID = Number(createCardAccountId)
    if (!Number.isInteger(accountID) || accountID <= 0) {
      setCreateCardState({ loading: false, error: 'Укажи корректный account_id.', success: '' })
      return
    }

    setCreateCardState({ loading: true, error: '', success: '' })
    setCreatedCard(null)

    try {
      const card = await cardsDomain.createMutation.mutateAsync(accountID)

      upsertCard(card)
      setCreatedCard(card)
      setCreateCardState({
        loading: false,
        error: '',
        success: 'Карта выпущена. CVV показывается только один раз.',
      })
    } catch (error) {
      setCreateCardState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to create card',
        success: '',
      })
    }
  }

  const requestCardRevealMFA = async () => {
    if (!requireToken(setCardRevealMfaState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardRevealMfaState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    setCardRevealMfaState({ loading: true, error: '', success: '' })

    try {
      await cardRevealMfaFlow.requestMutation.mutateAsync({
        purpose: 'card_reveal',
        card_id: cardID,
      })

      setCardRevealMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для показа реквизитов карты отправлен.',
      })
    } catch (error) {
      setCardRevealMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const revealCardNumber = async () => {
    if (!requireToken(setCardRevealState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardRevealState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    setCardRevealState({ loading: true, error: '', success: '' })

    try {
      const card = await cardsDomain.revealMutation.mutateAsync({
        cardID,
        body: { mfa_code: cardRevealMfaCode },
      })

      updateCardListWithoutRevealedNumber(card)
      setSelectedCardId(String(card.id))
      setSelectedCard((current) => current ? { ...current, ...card, number: undefined } : { ...card, number: undefined })
      setRevealedCardDetails(card)
      setCardRevealMfaCode('')
      setCardRevealState({ loading: false, error: '', success: 'Секретные реквизиты карты показаны.' })
    } catch (error) {
      setCardRevealState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to reveal card',
        success: '',
      })
    }
  }

  const requestCardPaymentMFA = async () => {
    if (!requireToken(setCardPaymentMfaState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardPaymentMfaState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    setCardPaymentMfaState({ loading: true, error: '', success: '' })

    try {
      await cardPaymentMfaFlow.requestMutation.mutateAsync({
        purpose: 'card_payment',
        card_id: cardID,
        amount: cardPaymentAmount,
      })

      setCardPaymentMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для оплаты картой отправлен.',
      })
    } catch (error) {
      setCardPaymentMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const handleCardPayment = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setCardPaymentState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardPaymentState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    const amountError = validateAmount(cardPaymentAmount)
    if (amountError) {
      setCardPaymentState({ loading: false, error: amountError, success: '' })
      return
    }

    setCardPaymentState({ loading: true, error: '', success: '' })
    setCardPaymentResult(null)

    try {
      const data = await cardsDomain.payMutation.mutateAsync({
        cardID,
        body: {
          amount: cardPaymentAmount,
          cvv: cardPaymentCVV,
          mfa_code: cardPaymentMfaCode,
          description: cardPaymentDescription,
        },
      })

      setCardPaymentResult(data)
      setCardOperationStatistics(null)
      setCardPaymentMfaCode('')
      setCardPaymentState({ loading: false, error: '', success: 'Оплата картой выполнена.' })
    } catch (error) {
      setCardPaymentState({
        loading: false,
        error: error instanceof Error ? error.message : 'Card payment failed',
        success: '',
      })
    }
  }

  const requestCardTransferMFA = async () => {
    if (!requireToken(setCardTransferMfaState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    const toCardNumber = cardTransferToCardNumber.trim()

    if (!cardID || !toCardNumber) {
      setCardTransferMfaState({
        loading: false,
        error: !cardID ? 'Выбери карту отправителя.' : 'Укажи номер карты получателя.',
        success: '',
      })
      return
    }

    setCardTransferMfaState({ loading: true, error: '', success: '' })

    try {
      await cardTransferMfaFlow.requestMutation.mutateAsync({
        purpose: 'card_transfer',
        card_id: cardID,
        to_card_number: toCardNumber,
        amount: cardTransferAmount,
      })

      setCardTransferMfaState({
        loading: false,
        error: '',
        success: 'MFA-код для перевода с карты отправлен.',
      })
    } catch (error) {
      setCardTransferMfaState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to request MFA code',
        success: '',
      })
    }
  }

  const handleCardTransfer = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setCardTransferState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    const toCardNumber = cardTransferToCardNumber.trim()

    if (!cardID || !toCardNumber) {
      setCardTransferState({
        loading: false,
        error: !cardID ? 'Выбери карту отправителя.' : 'Укажи номер карты получателя.',
        success: '',
      })
      return
    }

    const validationError = validateAmount(cardTransferAmount)
    if (validationError) {
      setCardTransferState({ loading: false, error: validationError, success: '' })
      return
    }

    if (cardTransferCVV.trim() === '' || cardTransferMfaCode.trim() === '') {
      setCardTransferState({ loading: false, error: 'Введи CVV и MFA code.', success: '' })
      return
    }

    setCardTransferResult(null)
    setCardTransferState(emptyState)
    setCardTransferConfirmOpen(true)
  }

  const confirmCardTransfer = async () => {
    if (!requireToken(setCardTransferState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    const toCardNumber = cardTransferToCardNumber.trim()

    if (!cardID || !toCardNumber) {
      setCardTransferConfirmOpen(false)
      setCardTransferState({
        loading: false,
        error: !cardID ? 'Выбери карту отправителя.' : 'Укажи номер карты получателя.',
        success: '',
      })
      return
    }

    setCardTransferState({ loading: true, error: '', success: '' })
    setCardTransferResult(null)

    try {
      const data = await cardsDomain.transferMutation.mutateAsync({
        cardID,
        body: {
          to_card_number: toCardNumber,
          amount: cardTransferAmount,
          cvv: cardTransferCVV,
          mfa_code: cardTransferMfaCode,
          description: cardTransferDescription,
        },
      })

      setCardTransferResult(data)
      setCardOperationStatistics(null)
      setCardTransferMfaCode('')
      setCardTransferState({ loading: false, error: '', success: 'Перевод с карты выполнен.' })
    } catch (error) {
      setCardTransferState({
        loading: false,
        error: error instanceof Error ? error.message : 'Card transfer failed',
        success: '',
      })
    }
  }

  const loadCardOperationStatistics = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()

    if (!requireToken(setCardStatisticsState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardStatisticsState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    const limit = Number(cardStatisticsLimit)
    if (!Number.isInteger(limit) || limit <= 0 || limit > 500) {
      setCardStatisticsState({ loading: false, error: 'Limit должен быть от 1 до 500.', success: '' })
      return
    }

    setCardStatisticsState({ loading: true, error: '', success: '' })
    setCardOperationStatistics(null)

    try {
      const data = await cardsDomain.operationStatisticsMutation.mutateAsync({ cardID, limit })

      setCardOperationStatistics(data)
      setCardStatisticsState({ loading: false, error: '', success: 'Статистика операций по карте загружена.' })
    } catch (error) {
      setCardStatisticsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load card operation statistics',
        success: '',
      })
    }
  }

  const closeCard = async () => {
    if (!requireToken(setCardCloseState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardCloseState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    setCloseConfirmOpen(true)
  }

  const confirmCloseCard = async () => {
    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCloseConfirmOpen(false)
      setCardCloseState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    setCardCloseState({ loading: true, error: '', success: '' })
    setCardCloseResult(null)

    try {
      const data = await cardsDomain.closeMutation.mutateAsync(cardID)

      setCardCloseResult(data)
      setCardOperationStatistics(null)
      applyClosedCard(data)
      setCloseConfirmOpen(false)
      setCardCloseState({ loading: false, error: '', success: 'Карта закрыта.' })
      showToast('Карта закрыта.', 'success')
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to close card'
      setCardCloseState({
        loading: false,
        error: message,
        success: '',
      })
      showToast(message, 'error')
    }
  }

  const hasOperationResult = cardPaymentResult || cardTransferResult
  const selectedCardSensitiveDetails =
    selectedCard && revealedCardDetails?.id === selectedCard.id ? revealedCardDetails : null

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader cardsPageHeader">
        <div>
          <h2>Карты пользователя</h2>
          <p>Выпуск, список, безопасный просмотр номера, оплата, перевод, закрытие и статистика операций по карте.</p>
        </div>

        <div className="actions">
          <Button type="button" onClick={loadCards} disabled={cardsState.loading || !token}>
            {cardsState.loading ? 'Загружаю...' : 'Загрузить карты'}
          </Button>
        </div>
      </div>

      <RequestStatus state={cardsState} />

      <div className="cardsWorkspace">
        <aside className="cardsColumn cardsNavigationColumn">
          <CardListPanel cards={cards} selectedCardId={selectedCardId} revealedCardDetails={selectedCardSensitiveDetails} onSelect={selectCard} />
        </aside>

        <main className="cardsColumn cardsOperationsColumn">
          <section className="subPanel cardsCreatePanel">
            <div className="subPanelHeader">
              <div>
                <h3>Выпуск карты</h3>
                <p className="mutedText">Создай новую карту для выбранного или введённого счёта.</p>
              </div>
            </div>

            <form className="cardsCreateInlineForm" onSubmit={createCard}>
              <label>
                <span>Account ID</span>
                <input
                  value={createCardAccountId}
                  onChange={(event) => setCreateCardAccountId(event.target.value)}
                  placeholder="ID счета"
                />
              </label>

              <Button type="submit" disabled={createCardState.loading || !token}>
                {createCardState.loading ? 'Выпускаю...' : 'Выпустить карту'}
              </Button>
            </form>

            <RequestStatus state={createCardState} />

            {createdCard && (
              <div className="result success compactResult">
                <strong>Карта выпущена</strong>
                <p className="mutedText">CVV показывается только один раз. Сохрани его для тестовых операций.</p>
                <pre>{JSON.stringify(formatCardForDisplay(createdCard), null, 2)}</pre>
              </div>
            )}
          </section>
          <section className="subPanel cardsOperationsPanel">
            <div className="subPanelHeader">
              <h3>Операции по карте</h3>
              {selectedCard && <span>card_id {selectedCard.id}</span>}
            </div>

            {!selectedCard && <div className="empty">Выбери карту слева, чтобы выполнить оплату, перевод или закрытие.</div>}

            {selectedCard && (
              <>
                {isCardClosed(selectedCard) && (
                  <div className="empty cardsClosedNotice">Карта закрыта. Операции по ней недоступны.</div>
                )}

                <div className="cardOperationsGrid">
                  <form className="actionBox" onSubmit={handleCardPayment}>
                    <h4>Оплата картой</h4>
                    <p>Сначала запроси MFA-код, потом выполни оплату.</p>

                    <label>
                      <span>Amount</span>
                      <input
                        value={cardPaymentAmount}
                        onChange={(event) => setCardPaymentAmount(event.target.value)}
                        disabled={isCardClosed(selectedCard)}
                      />
                    </label>
                    <label>
                      <span>Description</span>
                      <input
                        value={cardPaymentDescription}
                        onChange={(event) => setCardPaymentDescription(event.target.value)}
                        disabled={isCardClosed(selectedCard)}
                      />
                    </label>

                    <Button
                      className="secondary"
                      type="button"
                      onClick={requestCardPaymentMFA}
                      disabled={cardPaymentMfaState.loading || isCardClosed(selectedCard)}
                    >
                      {cardPaymentMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                    </Button>
                    <RequestStatus state={cardPaymentMfaState} />

                    <div className="cardsInlineFields">
                      <label>
                        <span>CVV</span>
                        <input
                          value={cardPaymentCVV}
                          onChange={(event) => setCardPaymentCVV(event.target.value)}
                          disabled={isCardClosed(selectedCard)}
                        />
                      </label>
                      <label>
                        <span>MFA code</span>
                        <input
                          value={cardPaymentMfaCode}
                          onChange={(event) => setCardPaymentMfaCode(event.target.value)}
                          disabled={isCardClosed(selectedCard)}
                        />
                      </label>
                    </div>

                    <Button type="submit" disabled={cardPaymentState.loading || isCardClosed(selectedCard)}>
                      {cardPaymentState.loading ? 'Оплачиваю...' : 'Оплатить'}
                    </Button>
                    <RequestStatus state={cardPaymentState} />
                  </form>

                  <form className="actionBox" onSubmit={handleCardTransfer}>
                    <h4>Перевод с карты</h4>
                    <p>Перевод идет с выбранной карты на карту-получатель.</p>

                    <label>
                      <span>To card number</span>
                      <input
                        value={cardTransferToCardNumber}
                        onChange={(event) => setCardTransferToCardNumber(event.target.value)}
                        placeholder="2200 0000 0000 0000"
                        disabled={isCardClosed(selectedCard)}
                      />
                    </label>
                    <div className="cardsInlineFields">
                      <label>
                        <span>Amount</span>
                        <input
                          value={cardTransferAmount}
                          onChange={(event) => setCardTransferAmount(event.target.value)}
                          disabled={isCardClosed(selectedCard)}
                        />
                      </label>
                      <label>
                        <span>Description</span>
                        <input
                          value={cardTransferDescription}
                          onChange={(event) => setCardTransferDescription(event.target.value)}
                          disabled={isCardClosed(selectedCard)}
                        />
                      </label>
                    </div>

                    <Button
                      className="secondary"
                      type="button"
                      onClick={requestCardTransferMFA}
                      disabled={cardTransferMfaState.loading || isCardClosed(selectedCard)}
                    >
                      {cardTransferMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                    </Button>
                    <RequestStatus state={cardTransferMfaState} />

                    <div className="cardsInlineFields">
                      <label>
                        <span>CVV</span>
                        <input
                          value={cardTransferCVV}
                          onChange={(event) => setCardTransferCVV(event.target.value)}
                          disabled={isCardClosed(selectedCard)}
                        />
                      </label>
                      <label>
                        <span>MFA code</span>
                        <input
                          value={cardTransferMfaCode}
                          onChange={(event) => setCardTransferMfaCode(event.target.value)}
                          disabled={isCardClosed(selectedCard)}
                        />
                      </label>
                    </div>

                    <Button type="submit" disabled={cardTransferState.loading || isCardClosed(selectedCard)}>
                      {cardTransferState.loading ? 'Перевожу...' : 'Перевести'}
                    </Button>
                    <RequestStatus state={cardTransferState} />
                  </form>                </div>
              </>
            )}
          </section>

          {hasOperationResult && (
            <section className="subPanel cardsResultsPanel">
              <div className="subPanelHeader">
                <h3>Результаты операций</h3>
              </div>
              {cardPaymentResult && (
                <div className="result success">
                  <strong>Результат оплаты</strong>
                  <pre>{JSON.stringify(cardPaymentResult, null, 2)}</pre>
                </div>
              )}
              {cardTransferResult && (
                <div className="result success">
                  <strong>Результат перевода</strong>
                  <pre>{JSON.stringify(cardTransferResult, null, 2)}</pre>
                </div>
              )}
            </section>
          )}

          {selectedCard && (
            <section className="subPanel cardsOperationStatisticsPanel">
              <OperationStatisticsPanel
                title="Статистика операций по карте"
                description="История и суммы по выбранной карте."
                endpointLabel="GET /cards/{cardId}/operations/statistics"
                limit={cardStatisticsLimit}
                state={cardStatisticsState}
                statistics={cardOperationStatistics}
                disabled={isCardClosed(selectedCard)}
                emptyText="Нажми “Получить статистику”, чтобы увидеть историю и суммы по выбранной карте."
                onLimitChange={setCardStatisticsLimit}
                onSubmit={loadCardOperationStatistics}
              />
            </section>
          )}
        </main>

        <aside className="cardsColumn cardsDetailsColumn">
          {selectedCard && (
            <section className="subPanel cardClosePanel dangerZone">
              <div className="subPanelHeader">
                <div>
                  <h3>Закрытие карты</h3>
                  <p className="mutedText">Закрытая карта больше не участвует в операциях.</p>
                </div>
                <span className={isCardClosed(selectedCard) ? 'badge mutedBadge' : 'badge successBadge'}>
                  {selectedCard.status}
                </span>
              </div>

              <Button
                className="danger"
                type="button"
                onClick={closeCard}
                disabled={cardCloseState.loading || isCardClosed(selectedCard)}
              >
                {cardCloseState.loading ? 'Закрываю...' : 'Закрыть карту'}
              </Button>
              <RequestStatus state={cardCloseState} />

              {cardCloseResult && (
                <div className="result success compactResult">
                  <strong>Результат закрытия карты</strong>
                  <pre>{JSON.stringify(cardCloseResult, null, 2)}</pre>
                </div>
              )}
            </section>
          )}

          {selectedCard && (
            <section className="subPanel cardRevealPanel">
              <div className="subPanelHeader">
                <h3>Секретные реквизиты карты</h3>
                <span className="badge mutedBadge">MFA</span>
              </div>

              <p className="mutedText">
                Полный PAN и срок действия показываются только через endpoint <code>POST /cards/{'{cardId}'}/reveal</code>. После MFA реквизиты отобразятся в выбранной карточке слева. CVV не показывается повторно.
              </p>

              <div className="cardRevealActions">
                <Button
                  className="secondary"
                  type="button"
                  onClick={requestCardRevealMFA}
                  disabled={cardRevealMfaState.loading || isCardClosed(selectedCard)}
                >
                  {cardRevealMfaState.loading ? 'Отправляю...' : 'Запросить MFA для реквизитов'}
                </Button>
                <RequestStatus state={cardRevealMfaState} />

                <label>
                  <span>MFA code</span>
                  <input
                    value={cardRevealMfaCode}
                    onChange={(event) => setCardRevealMfaCode(event.target.value)}
                    disabled={isCardClosed(selectedCard)}
                  />
                </label>

                <Button type="button" onClick={revealCardNumber} disabled={cardRevealState.loading || isCardClosed(selectedCard)}>
                  {cardRevealState.loading ? 'Показываю...' : 'Показать реквизиты'}
                </Button>
                <RequestStatus state={cardRevealState} />
              </div>
            </section>
          )}
        </aside>
      </div>

      <OperationConfirmDialog
        open={cardTransferConfirmOpen}
        title="Подтвердить перевод с карты"
        message="Проверь детали операции перед выполнением."
        confirmText="Выполнить перевод"
        loading={cardTransferState.loading}
        error={cardTransferState.error}
        result={
          cardTransferResult ? (
            <div className="operationDialogResultGrid">
              <div><span>Status</span><strong>{cardTransferResult.status}</strong></div>
              <div><span>Transaction ID</span><strong>{cardTransferResult.transaction_id}</strong></div>
              <div><span>From card ID</span><strong>{cardTransferResult.from_card_id}</strong></div>
              <div><span>To card ID</span><strong>{cardTransferResult.to_card_id || '-'}</strong></div>
              <div><span>Amount</span><strong>{cardTransferResult.amount} RUB</strong></div>
              <div><span>Recipient</span><strong>{formatCardNumber(cardTransferToCardNumber)}</strong></div>
            </div>
          ) : undefined
        }
        onConfirm={() => void confirmCardTransfer()}
        onClose={() => setCardTransferConfirmOpen(false)}
      >
        <div className="operationConfirmDetails">
          <div><span>Операция</span><strong>Перевод с карты на карту</strong></div>
          <div><span>Карта отправителя</span><strong>{selectedCard ? formatCardNumber(selectedCard.number || selectedCard.masked_number) : '-'}</strong></div>
          <div><span>Карта получателя</span><strong>{formatCardNumber(cardTransferToCardNumber)}</strong></div>
          <div><span>Сумма</span><strong>{cardTransferAmount} RUB</strong></div>
          <div><span>Описание</span><strong>{cardTransferDescription || '-'}</strong></div>
          <div><span>Статус</span><strong>{cardTransferResult ? cardTransferResult.status : 'Ожидает подтверждения'}</strong></div>
        </div>
      </OperationConfirmDialog>

      <ConfirmDialog
        open={closeConfirmOpen}
        title="Закрыть карту"
        message="Закрыть выбранную карту? Операция необратима."
        confirmText="Закрыть карту"
        danger
        loading={cardCloseState.loading}
        onConfirm={() => void confirmCloseCard()}
        onCancel={() => setCloseConfirmOpen(false)}
      />
    </Card>
  )
}
