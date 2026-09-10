# Keep JNI methods and native class
-keepclasseswithmembernames class * {
    native <methods>;
}

-keep class com.openflux.client.core.** { *; }
-keep class com.openflux.client.service.** { *; }
-keep class com.openflux.client.data.** { *; }

# Material & ViewBinding
-keepclassmembers class * implements androidx.viewbinding.ViewBinding {
    public static *** inflate(...);
    public static *** bind(...);
}

# Coroutines
-dontwarn kotlinx.coroutines.**
