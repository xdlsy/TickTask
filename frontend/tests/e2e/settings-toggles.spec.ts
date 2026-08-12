import { test, expect } from '../support/fixtures'

test.describe('@p2 Settings · 开关与滑块往返', () => {
  test('SET-UI-002: 翻转 3 开关 + 调缓冲比例 → 保存 → 重载持久 → 还原', async ({ page }) => {
    await page.goto('/settings')

    const switches = page.locator('.el-switch')
    const thumb = page.locator('.el-slider__button')
    const save = () => page.getByRole('button', { name: '保存设置' }).first().click()

    const beforeSw = await switches.evaluateAll((els) => els.map((e) => String(e.classList.contains('is-checked'))))
    const beforeSlider = (await thumb.getAttribute('aria-valuenow')) || '15'

    for (let i = 0; i < 3; i++) await switches.nth(i).click()
    const runway = page.locator('.el-slider__runway')
    const box = await runway.boundingBox()
    if (box) await runway.click({ position: { x: box.width * 0.8, y: box.height / 2 } })

    await save()

    await page.reload()
    const swAfter = page.locator('.el-switch')
    const thumbAfter = page.locator('.el-slider__button')
    const afterSw = await swAfter.evaluateAll((els) => els.map((e) => String(e.classList.contains('is-checked'))))
    for (let i = 0; i < 3; i++) expect(afterSw[i]).not.toBe(beforeSw[i])
    const afterSlider = await thumbAfter.getAttribute('aria-valuenow')
    expect(afterSlider).not.toBe(beforeSlider)

    // 还原:再翻 3 开关 + 滑块回到低端
    for (let i = 0; i < 3; i++) await swAfter.nth(i).click()
    const rb = await page.locator('.el-slider__runway').boundingBox()
    if (rb) await page.locator('.el-slider__runway').click({ position: { x: rb.width * 0.3, y: rb.height / 2 } })
    await save()
  })
})