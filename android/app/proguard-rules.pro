# ============================================================================
# OpenFlux R8 / ProGuard Configuration
# ============================================================================

# JNI bridge to native Go core
-keepclasseswithmembernames,includedescriptorclasses class * {
    native <methods>;
}

-keep class com.openflux.client.core.OpenFluxCore {
    public <methods>;
    native <methods>;
}

# Android Services registered in Manifest
-keep class com.openflux.client.service.OpenFluxVpnService { *; }
-keep class com.openflux.client.service.OpenFluxProxyService { *; }
-keep class com.openflux.client.service.OpenFluxTileService { *; }

# Data classes & preferences
-keepclassmembers class com.openflux.client.data.AppPreferences {
    public <methods>;
}

-keepclassmembers class com.openflux.client.data.SecureSettings {
    public <methods>;
}

-keep class com.openflux.client.data.QrConfigData { *; }

# ViewBinding
-keepclassmembers class * implements androidx.viewbinding.ViewBinding {
    public static *** inflate(...);
    public static *** bind(...);
}

# R8 Optimizations
-allowaccessmodification
-repackageclasses 'o'

# Strip debug logs in release builds
-assumenosideeffects class android.util.Log {
    public static *** d(...);
    public static *** v(...);
    public static *** i(...);
}

# Keep line numbers and annotations for readable crash traces
-renamesourcefileattribute SourceFile
-keepattributes SourceFile,LineNumberTable,InnerClasses,EnclosingMethod,*Annotation*,Signature

# Library-specific warning suppressions
-dontwarn kotlinx.coroutines.**
-dontwarn com.google.zxing.**
