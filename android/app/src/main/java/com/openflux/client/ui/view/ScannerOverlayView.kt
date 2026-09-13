package com.openflux.client.ui.view

import android.content.Context
import android.graphics.Canvas
import android.graphics.Color
import android.graphics.Paint
import android.graphics.PorterDuff
import android.graphics.PorterDuffXfermode
import android.graphics.RectF
import android.util.AttributeSet
import android.view.View

class ScannerOverlayView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0
) : View(context, attrs, defStyleAttr) {

    private val scrimPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = Color.parseColor("#99000000") // 60% translucent black scrim
    }

    private val eraserPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        xfermode = PorterDuffXfermode(PorterDuff.Mode.CLEAR)
    }

    private val borderPaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        color = Color.parseColor("#CCFFFFFF") // Crisp white rounded frame
        style = Paint.Style.STROKE
        strokeWidth = 3f * resources.displayMetrics.density
    }

    val frameRect = RectF()
    private val cornerRadius = 24f * resources.displayMetrics.density

    init {
        setLayerType(LAYER_TYPE_HARDWARE, null)
    }

    override fun onSizeChanged(w: Int, h: Int, oldw: Int, oldh: Int) {
        super.onSizeChanged(w, h, oldw, oldh)
        val boxSize = (minOf(w, h) * 0.72f).coerceIn(
            220f * resources.displayMetrics.density,
            300f * resources.displayMetrics.density
        )
        val left = (w - boxSize) / 2f
        // Slightly above vertical center for ergonomic scanning (as in screenshot)
        val top = (h - boxSize) / 2f - 20f * resources.displayMetrics.density
        frameRect.set(left, top, left + boxSize, top + boxSize)
    }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        // 1. Draw dark translucent scrim
        canvas.drawRect(0f, 0f, width.toFloat(), height.toFloat(), scrimPaint)

        // 2. Clear out the transparent rounded box
        canvas.drawRoundRect(frameRect, cornerRadius, cornerRadius, eraserPaint)

        // 3. Draw rounded frame border
        canvas.drawRoundRect(frameRect, cornerRadius, cornerRadius, borderPaint)
    }
}
