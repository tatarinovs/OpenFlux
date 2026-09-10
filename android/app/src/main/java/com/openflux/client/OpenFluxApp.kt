package com.openflux.client

import android.app.Application
import android.util.Log
import java.io.File
import java.io.PrintWriter
import java.io.StringWriter

class OpenFluxApp : Application() {

    override fun onCreate() {
        super.onCreate()

        val defaultHandler = Thread.getDefaultUncaughtExceptionHandler()
        Thread.setDefaultUncaughtExceptionHandler { thread, throwable ->
            try {
                val sw = StringWriter()
                val pw = PrintWriter(sw)
                throwable.printStackTrace(pw)
                val stackTrace = sw.toString()
                Log.e("OpenFluxCrash", "FATAL CRASH on thread ${thread.name}:\n$stackTrace")

                val crashFile = File(filesDir, "crash.log")
                crashFile.writeText("CRASH TIME: ${java.util.Date()}\nTHREAD: ${thread.name}\n\n$stackTrace")
            } catch (e: Exception) {
                e.printStackTrace()
            }
            defaultHandler?.uncaughtException(thread, throwable)
        }
    }
}
