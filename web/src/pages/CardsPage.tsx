import { useEffect, useState } from 'react'
import type { FormEvent } from 'react'
import { RequestStatus } from '../components/RequestStatus'
import type { CardPaymentResponse, CardResponse, CardTransferResponse, CloseCardResponse } from '../types/card'
import { emptyState, type RequestState } from '../types/common'
import {
  formatDate,
  getCardBadgeClass,
  getCardDisplayNumber,
  getCardStatusText,
  isCardClosed,
} from '../utils/format'
import { Button } from '../components/ui/Button'
import { CardListPanel } from '../features/cards/CardListPanel'
import { Card } from '../components/ui/Card'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { useCards } from '../hooks/useCards'
import { useMfaFlow } from '../hooks/useMfaFlow'
import { useToast } from '../hooks/useToast'
import { validateAmount, validatePositiveInteger } from '../utils/validation'

type CardsPageProps = {
  token: string
  sharedAccountId: string
}

export function CardsPage({ token, sharedAccountId }: CardsPageProps) {
  const { showToast } = useToast()
  const cardsDomain = useCards(token)
  const cardPaymentMfaFlow = useMfaFlow(token)
  const cardTransferMfaFlow = useMfaFlow(token)
  const [cardsState, setCardsState] = useState<RequestState>(emptyState)
  const [createCardState, setCreateCardState] = useState<RequestState>(emptyState)
  const [cardDetailsState, setCardDetailsState] = useState<RequestState>(emptyState)
  const [cardPaymentMfaState, setCardPaymentMfaState] = useState<RequestState>(emptyState)
  const [cardPaymentState, setCardPaymentState] = useState<RequestState>(emptyState)
  const [cardTransferMfaState, setCardTransferMfaState] = useState<RequestState>(emptyState)
  const [cardTransferState, setCardTransferState] = useState<RequestState>(emptyState)
  const [cardCloseState, setCardCloseState] = useState<RequestState>(emptyState)

  const [cards, setCards] = useState<CardResponse[]>([])
  const [selectedCardId, setSelectedCardId] = useState('')
  const [selectedCard, setSelectedCard] = useState<CardResponse | null>(null)
  const [createCardAccountId, setCreateCardAccountId] = useState(sharedAccountId)
  const [createdCard, setCreatedCard] = useState<CardResponse | null>(null)

  const [cardPaymentAmount, setCardPaymentAmount] = useState('100.00')
  const [cardPaymentCVV, setCardPaymentCVV] = useState('')
  const [cardPaymentMfaCode, setCardPaymentMfaCode] = useState('')
  const [cardPaymentDescription, setCardPaymentDescription] = useState('Card payment')

  const [cardTransferToCardId, setCardTransferToCardId] = useState('')
  const [cardTransferAmount, setCardTransferAmount] = useState('100.00')
  const [cardTransferCVV, setCardTransferCVV] = useState('')
  const [cardTransferMfaCode, setCardTransferMfaCode] = useState('')
  const [cardTransferDescription, setCardTransferDescription] = useState('Card transfer')

  const [cardPaymentResult, setCardPaymentResult] = useState<CardPaymentResponse | null>(null)
  const [cardTransferResult, setCardTransferResult] = useState<CardTransferResponse | null>(null)
  const [cardCloseResult, setCardCloseResult] = useState<CloseCardResponse | null>(null)
  const [closeConfirmOpen, setCloseConfirmOpen] = useState(false)

  useEffect(() => {
    if (sharedAccountId && !createCardAccountId) {
      setCreateCardAccountId(sharedAccountId)
    }
  }, [sharedAccountId, createCardAccountId])

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
    setCardCloseResult(null)
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

  const loadCardDetails = async () => {
    if (!requireToken(setCardDetailsState)) {
      return
    }

    const cardID = selectedCardIDNumber()
    if (!cardID) {
      setCardDetailsState({ loading: false, error: 'Выбери карту.', success: '' })
      return
    }

    setCardDetailsState({ loading: true, error: '', success: '' })

    try {
      const card = await cardsDomain.detailMutation.mutateAsync(cardID)
      upsertCard(card)
      setCardDetailsState({ loading: false, error: '', success: 'Данные карты обновлены.' })
    } catch (error) {
      setCardDetailsState({
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load card',
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
    const toCardID = Number(cardTransferToCardId)

    if (!cardID || !Number.isInteger(toCardID) || toCardID <= 0) {
      setCardTransferMfaState({
        loading: false,
        error: !cardID ? 'Выбери карту отправителя.' : 'Укажи корректный to_card_id.',
        success: '',
      })
      return
    }

    setCardTransferMfaState({ loading: true, error: '', success: '' })

    try {
      await cardTransferMfaFlow.requestMutation.mutateAsync({
        purpose: 'card_transfer',
        card_id: cardID,
        to_card_id: toCardID,
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
    const toCardID = Number(cardTransferToCardId)

    if (!cardID || !Number.isInteger(toCardID) || toCardID <= 0) {
      setCardTransferState({
        loading: false,
        error: !cardID ? 'Выбери карту отправителя.' : 'Укажи корректный to_card_id.',
        success: '',
      })
      return
    }

    const validationError = validatePositiveInteger(cardTransferToCardId, 'To card ID') || validateAmount(cardTransferAmount)
    if (validationError) {
      setCardTransferState({ loading: false, error: validationError, success: '' })
      return
    }

    setCardTransferState({ loading: true, error: '', success: '' })
    setCardTransferResult(null)

    try {
      const data = await cardsDomain.transferMutation.mutateAsync({
        cardID,
        body: {
        to_card_id: toCardID,
        amount: cardTransferAmount,
        cvv: cardTransferCVV,
        mfa_code: cardTransferMfaCode,
        description: cardTransferDescription,
        },
      })

      setCardTransferResult(data)
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

  return (
    <Card variant="plain" className="panel">
      <div className="panelHeader">
        <div>
          <h2>Карты пользователя</h2>
          <p>Все действия с картами: выпуск, список, просмотр, оплата, перевод и закрытие.</p>
        </div>

        <div className="actions">
          <Button type="button" onClick={loadCards} disabled={cardsState.loading || !token}>
            {cardsState.loading ? 'Загружаю...' : 'Загрузить карты'}
          </Button>
        </div>
      </div>

      <RequestStatus state={cardsState} />

      <div className="cardsLayout">
        <section className="subPanel">
          <div className="subPanelHeader"><h3>Выпуск карты</h3></div>

          <form className="form" onSubmit={createCard}>
            <label>
              <span>Account ID</span>
              <input value={createCardAccountId} onChange={(event) => setCreateCardAccountId(event.target.value)} placeholder="ID счета" />
            </label>

            <Button type="submit" disabled={createCardState.loading || !token}>
              {createCardState.loading ? 'Выпускаю...' : 'Выпустить карту'}
            </Button>
          </form>

          <RequestStatus state={createCardState} />

          {createdCard && (
            <div className="result success">
              <strong>Карта выпущена</strong>
              <p className="mutedText">CVV показывается только один раз. Сохрани его для тестовых операций.</p>
              <pre>{JSON.stringify(createdCard, null, 2)}</pre>
            </div>
          )}
        </section>

        <CardListPanel
          cards={cards}
          selectedCardId={selectedCardId}
          onSelect={selectCard}
        />

        <section className="subPanel cardDetailsPanel">
          <div className="subPanelHeader">
            <h3>Выбранная карта</h3>
            {selectedCard && <span className={getCardBadgeClass(selectedCard)}>{getCardStatusText(selectedCard)}</span>}
          </div>

          {!selectedCard && <div className="empty">Выбери карту из списка.</div>}

          {selectedCard && (
            <>
              <div className="detailsGrid">
                <div><span>ID</span><strong>{selectedCard.id}</strong></div>
                <div><span>Account ID</span><strong>{selectedCard.account_id}</strong></div>
                <div><span>Number</span><strong>{getCardDisplayNumber(selectedCard)}</strong></div>
                <div><span>Expiry</span><strong>{selectedCard.expiry || '-'}</strong></div>
                <div><span>Status</span><strong>{selectedCard.status}</strong></div>
                <div><span>Created</span><strong>{formatDate(selectedCard.created_at)}</strong></div>
                <div><span>Closed at</span><strong>{formatDate(selectedCard.closed_at)}</strong></div>
              </div>

              <div className="actions topGap">
                <Button className="secondary" type="button" onClick={loadCardDetails} disabled={cardDetailsState.loading}>
                  {cardDetailsState.loading ? 'Обновляю...' : 'Показать детали'}
                </Button>
              </div>

              <RequestStatus state={cardDetailsState} />

              <div className="cardActionsGrid">
                <form className="actionBox" onSubmit={handleCardPayment}>
                  <h4>Оплата картой</h4>
                  <p>Сначала запроси MFA-код, потом выполни оплату.</p>

                  <label><span>Amount</span><input value={cardPaymentAmount} onChange={(event) => setCardPaymentAmount(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>
                  <label><span>Description</span><input value={cardPaymentDescription} onChange={(event) => setCardPaymentDescription(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>

                  <Button className="secondary" type="button" onClick={requestCardPaymentMFA} disabled={cardPaymentMfaState.loading || isCardClosed(selectedCard)}>
                    {cardPaymentMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                  </Button>
                  <RequestStatus state={cardPaymentMfaState} />

                  <label><span>CVV</span><input value={cardPaymentCVV} onChange={(event) => setCardPaymentCVV(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>
                  <label><span>MFA code</span><input value={cardPaymentMfaCode} onChange={(event) => setCardPaymentMfaCode(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>

                  <Button type="submit" disabled={cardPaymentState.loading || isCardClosed(selectedCard)}>
                    {cardPaymentState.loading ? 'Оплачиваю...' : 'Оплатить'}
                  </Button>
                  <RequestStatus state={cardPaymentState} />
                </form>

                <form className="actionBox" onSubmit={handleCardTransfer}>
                  <h4>Перевод с карты</h4>
                  <p>Перевод идет с выбранной карты на карту-получатель.</p>

                  <label><span>To card ID</span><input value={cardTransferToCardId} onChange={(event) => setCardTransferToCardId(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>
                  <label><span>Amount</span><input value={cardTransferAmount} onChange={(event) => setCardTransferAmount(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>
                  <label><span>Description</span><input value={cardTransferDescription} onChange={(event) => setCardTransferDescription(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>

                  <Button className="secondary" type="button" onClick={requestCardTransferMFA} disabled={cardTransferMfaState.loading || isCardClosed(selectedCard)}>
                    {cardTransferMfaState.loading ? 'Отправляю...' : 'Запросить MFA'}
                  </Button>
                  <RequestStatus state={cardTransferMfaState} />

                  <label><span>CVV</span><input value={cardTransferCVV} onChange={(event) => setCardTransferCVV(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>
                  <label><span>MFA code</span><input value={cardTransferMfaCode} onChange={(event) => setCardTransferMfaCode(event.target.value)} disabled={isCardClosed(selectedCard)} /></label>

                  <Button type="submit" disabled={cardTransferState.loading || isCardClosed(selectedCard)}>
                    {cardTransferState.loading ? 'Перевожу...' : 'Перевести'}
                  </Button>
                  <RequestStatus state={cardTransferState} />
                </form>

                <div className="actionBox dangerZone">
                  <h4>Закрытие карты</h4>
                  <p>Закрытая карта больше не участвует в операциях.</p>
                  <Button className="danger" type="button" onClick={closeCard} disabled={cardCloseState.loading || isCardClosed(selectedCard)}>
                    {cardCloseState.loading ? 'Закрываю...' : 'Закрыть карту'}
                  </Button>
                  <RequestStatus state={cardCloseState} />
                </div>
              </div>

              {cardPaymentResult && <div className="result success"><strong>Результат оплаты</strong><pre>{JSON.stringify(cardPaymentResult, null, 2)}</pre></div>}
              {cardTransferResult && <div className="result success"><strong>Результат перевода</strong><pre>{JSON.stringify(cardTransferResult, null, 2)}</pre></div>}
              {cardCloseResult && <div className="result success"><strong>Результат закрытия карты</strong><pre>{JSON.stringify(cardCloseResult, null, 2)}</pre></div>}
            </>
          )}
        </section>
      </div>
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
